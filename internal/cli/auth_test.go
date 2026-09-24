package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/auth"
	clierrors "github.com/thedavidweng/monarchmoney-cli/internal/errors"
)

func withAuthTestDefaults(t *testing.T, sessionPath string) func() {
	t.Helper()

	oldPath := defaultSessionPath
	oldAuthenticate := authenticateSession
	oldReadPassword := readPassword
	oldScanInput := scanInput
	oldFetchIdentity := fetchIdentity
	oldExitFunc := exitFunc
	oldJSONMode := jsonMode
	oldPretty := pretty
	oldProfile := profile

	defaultSessionPath = func() string { return sessionPath }
	authenticateSession = auth.Authenticate
	readPassword = func(int) ([]byte, error) {
		return nil, errors.New("unexpected password prompt")
	}
	scanInput = func(...any) (int, error) {
		return 0, errors.New("unexpected prompt")
	}
	fetchIdentity = func(_ context.Context, _ string) (*identityResult, error) {
		return &identityResult{Email: "fallback@example.com"}, nil
	}
	exitFunc = func(int) {}
	jsonMode = false
	pretty = false
	profile = "default"

	return func() {
		defaultSessionPath = oldPath
		authenticateSession = oldAuthenticate
		readPassword = oldReadPassword
		scanInput = oldScanInput
		fetchIdentity = oldFetchIdentity
		exitFunc = oldExitFunc
		jsonMode = oldJSONMode
		pretty = oldPretty
		profile = oldProfile
	}
}

func TestLoginUsesPasswordFlagWithoutPrompt(t *testing.T) {
	sessionPath := filepath.Join(t.TempDir(), "session.json")
	restore := withAuthTestDefaults(t, sessionPath)
	defer restore()

	sawPasswordPrompt := false
	readPassword = func(int) ([]byte, error) {
		sawPasswordPrompt = true
		return nil, errors.New("should not prompt")
	}

	called := false
	var gotEmail, gotPassword string
	authenticateSession = func(email, password, mfaCode, mfaSecret string) (*auth.Session, error) {
		called = true
		gotEmail = email
		gotPassword = password
		return &auth.Session{
			Email:     email,
			Token:     "token-123",
			CreatedAt: time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC),
		}, nil
	}

	if err := loginCmd.Flags().Set("email", "a@example.com"); err != nil {
		t.Fatalf("Set email flag error = %v", err)
	}
	if err := loginCmd.Flags().Set("password", "secret"); err != nil {
		t.Fatalf("Set password flag error = %v", err)
	}
	_ = loginCmd.Flags().Set("mfa-code", "")
	_ = loginCmd.Flags().Set("mfa-secret", "")

	out := captureStdout(t, func() {
		loginCmd.Run(loginCmd, nil)
	})

	if !called {
		t.Fatal("authenticateSession was not called")
	}
	if gotEmail != "a@example.com" || gotPassword != "secret" {
		t.Fatalf("authenticateSession args = %q %q", gotEmail, gotPassword)
	}
	if sawPasswordPrompt {
		t.Fatal("password prompt was triggered even though --password was provided")
	}
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v; output=%q", err, out)
	}
	var saved auth.Session
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if saved.Email != "a@example.com" || saved.Token != "token-123" {
		t.Fatalf("saved session = %#v", saved)
	}
}

func TestLoginJSONIncludesSessionDetails(t *testing.T) {
	sessionPath := filepath.Join(t.TempDir(), "session.json")
	restore := withAuthTestDefaults(t, sessionPath)
	defer restore()

	jsonMode = true
	authenticateSession = func(email, password, mfaCode, mfaSecret string) (*auth.Session, error) {
		return &auth.Session{
			Email:     email,
			Token:     "token-123",
			CreatedAt: time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC),
		}, nil
	}

	_ = loginCmd.Flags().Set("email", "a@example.com")
	_ = loginCmd.Flags().Set("password", "secret")
	_ = loginCmd.Flags().Set("mfa-code", "")
	_ = loginCmd.Flags().Set("mfa-secret", "")

	out := captureStdout(t, func() {
		loginCmd.Run(loginCmd, nil)
	})

	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Status      string    `json:"status"`
			Email       string    `json:"email"`
			SessionPath string    `json:"session_path"`
			CreatedAt   time.Time `json:"created_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(trimNewline(out)), &env); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output=%q", err, out)
	}
	if !env.OK || env.Data.Status != "logged in" || env.Data.Email != "a@example.com" || env.Data.SessionPath != sessionPath {
		t.Fatalf("login JSON = %#v", env)
	}
}

func TestAuthStatus(t *testing.T) {
	t.Run("success", testAuthStatusSuccess)
	t.Run("missing session", testAuthStatusMissingSession)
	t.Run("expired session", testAuthStatusExpiredSession)
	t.Run("network error", testAuthStatusNetworkError)
}

func testAuthStatusSuccess(t *testing.T) {
	sessionPath := filepath.Join(t.TempDir(), "session.json")
	restore := withAuthTestDefaults(t, sessionPath)
	defer restore()
	jsonMode = true

	store := auth.NewStore(sessionPath)
	if err := store.Save(&auth.Session{
		Profile:   "default",
		Email:     "a@example.com",
		Token:     "token-123",
		CreatedAt: time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	gotToken := ""
	fetchIdentity = func(_ context.Context, token string) (*identityResult, error) {
		gotToken = token
		return &identityResult{Email: "a@example.com"}, nil
	}

	out := captureStdout(t, func() {
		statusCmd.Run(statusCmd, nil)
	})

	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Authenticated bool   `json:"authenticated"`
			SessionValid  bool   `json:"session_valid"`
			Email         string `json:"email"`
			SessionPath   string `json:"session_path"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(trimNewline(out)), &env); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output=%q", err, out)
	}
	if gotToken != "token-123" {
		t.Fatalf("token = %q, want token-123", gotToken)
	}
	if !env.OK || !env.Data.Authenticated || !env.Data.SessionValid || env.Data.Email != "a@example.com" || env.Data.SessionPath != sessionPath {
		t.Fatalf("status JSON = %#v", env)
	}
}

func testAuthStatusMissingSession(t *testing.T) {
	sessionPath := filepath.Join(t.TempDir(), "missing.json")
	restore := withAuthTestDefaults(t, sessionPath)
	defer restore()

	jsonMode = true
	var exitCode int
	exitFunc = func(code int) {
		exitCode = code
	}
	called := false
	fetchIdentity = func(context.Context, string) (*identityResult, error) {
		called = true
		return nil, nil
	}

	out := captureStdout(t, func() {
		statusCmd.Run(statusCmd, nil)
	})

	var env struct {
		OK    bool `json:"ok"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(trimNewline(out)), &env); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output=%q", err, out)
	}
	if exitCode != 3 || env.Error.Code != string(clierrors.AuthRequired) {
		t.Fatalf("missing session = exitCode %d, env %#v", exitCode, env)
	}
	if called {
		t.Fatal("fetchIdentity should not be called when the session file is missing")
	}
}

func testAuthStatusExpiredSession(t *testing.T) {
	sessionPath := filepath.Join(t.TempDir(), "session.json")
	restore := withAuthTestDefaults(t, sessionPath)
	defer restore()

	store := auth.NewStore(sessionPath)
	if err := store.Save(&auth.Session{
		Profile:   "default",
		Email:     "a@example.com",
		Token:     "token-123",
		CreatedAt: time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	jsonMode = true
	var exitCode int
	exitFunc = func(code int) {
		exitCode = code
	}
	fetchIdentity = func(context.Context, string) (*identityResult, error) {
		return nil, clierrors.New(clierrors.AuthSessionExpired, "session token expired or invalid; run `monarch auth login` again", clierrors.CatAuth, true, nil)
	}

	out := captureStdout(t, func() {
		statusCmd.Run(statusCmd, nil)
	})

	var env struct {
		OK    bool `json:"ok"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(trimNewline(out)), &env); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output=%q", err, out)
	}
	if exitCode != 3 || env.Error.Code != string(clierrors.AuthSessionExpired) {
		t.Fatalf("expired session = exitCode %d, env %#v", exitCode, env)
	}
	if env.Error.Message == "" {
		t.Fatal("expired session message empty, want guidance")
	}
}

func testAuthStatusNetworkError(t *testing.T) {
	sessionPath := filepath.Join(t.TempDir(), "session.json")
	restore := withAuthTestDefaults(t, sessionPath)
	defer restore()

	store := auth.NewStore(sessionPath)
	if err := store.Save(&auth.Session{
		Profile:   "default",
		Email:     "a@example.com",
		Token:     "token-123",
		CreatedAt: time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	jsonMode = true
	var exitCode int
	exitFunc = func(code int) {
		exitCode = code
	}
	fetchIdentity = func(context.Context, string) (*identityResult, error) {
		return nil, clierrors.New(clierrors.NetworkUnreachable, "failed to reach Monarch API", clierrors.CatNetwork, true, nil)
	}

	out := captureStdout(t, func() {
		statusCmd.Run(statusCmd, nil)
	})

	var env struct {
		OK    bool `json:"ok"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(trimNewline(out)), &env); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output=%q", err, out)
	}
	if exitCode != 5 || env.Error.Code != string(clierrors.NetworkUnreachable) {
		t.Fatalf("network error = exitCode %d, env %#v", exitCode, env)
	}
}
