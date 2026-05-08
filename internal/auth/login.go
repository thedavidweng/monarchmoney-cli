package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/monarchmoney-cli/monarch/internal/errors"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
	// Add other fields if needed for UserID/HouseholdID
}

// Authenticate performs login against the Monarch API.
func Authenticate(email, password string) (*Session, error) {
	// For Phase 2, we just implement the client logic. 
	// Actual network calls will be refined as we build the full client.
	
	reqBody := loginRequest{
		Username: email,
		Password: password,
	}
	body, _ := json.Marshal(reqBody)

	// Mock/Actual endpoint from PRD: https://api.monarch.com/auth/login/
	req, err := http.NewRequest("POST", "https://api.monarch.com/auth/login/", bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.New(errors.InternalError, "failed to create login request", errors.CatInternal, false, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Platform", "web")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New(errors.NetworkUnreachable, "failed to reach Monarch API", errors.CatNetwork, true, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		// Potential MFA check here
		return nil, errors.New(errors.AuthRequired, "invalid credentials or MFA required", errors.CatAuth, false, nil)
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
