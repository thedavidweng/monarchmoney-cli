package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
)

var loginEndpoint = "https://api.monarch.com/auth/login/"
var newLoginHTTPClient = func() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

type loginRequest struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	SupportsMFA   bool   `json:"supports_mfa"`
	TrustedDevice bool   `json:"trusted_device"`
	TOTP          string `json:"totp,omitempty"`
}

type loginResponse struct {
	Token           string      `json:"token"`
	TokenExpiration interface{} `json:"tokenExpiration"`
	// Add other fields if needed for UserID/HouseholdID
}

// Authenticate performs login against the Monarch API.
func Authenticate(email, password, mfaCode, mfaSecret string) (*Session, error) {
	if mfaSecret != "" {
		code, err := totp.GenerateCode(mfaSecret, time.Now())
		if err != nil {
			return nil, errors.New(errors.InternalError, "failed to generate MFA code", errors.CatInternal, false, err)
		}
		mfaCode = code
	}

	reqBody := loginRequest{
		Username:      email,
		Password:      password,
		SupportsMFA:   true,
		TrustedDevice: true,
		TOTP:          mfaCode,
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", loginEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.New(errors.InternalError, "failed to create login request", errors.CatInternal, false, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Platform", "web")

	client := newLoginHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New(errors.NetworkUnreachable, "failed to reach Monarch API", errors.CatNetwork, true, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 401 {
		if mfaCode == "" && mfaSecret == "" {
			return nil, errors.New(errors.AuthMFARequired, "MFA code required", errors.CatAuth, false, nil)
		}
		return nil, errors.New(errors.AuthMFAInvalid, "invalid credentials or MFA code", errors.CatAuth, false, nil)
	}

	if resp.StatusCode != 200 {
		return nil, errors.New(errors.APIError, fmt.Sprintf("API returned status %d", resp.StatusCode), errors.CatAPI, false, nil)
	}

	var loginResp loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, errors.New(errors.APISchemaChanged, "failed to parse login response", errors.CatAPI, false, err)
	}

	return &Session{
		Token:     loginResp.Token,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}
