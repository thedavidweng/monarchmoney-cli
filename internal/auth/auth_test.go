package auth

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "session", "session.json"))
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	sess := &Session{
		Profile:     "default",
		CreatedAt:   now,
		UpdatedAt:   now,
		Token:       "token-123",
		UserID:      "user-1",
		HouseholdID: "house-1",
	}

	if err := store.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(store.Path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("file perm = %v, want 0600", got)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Token != sess.Token || loaded.Profile != sess.Profile || loaded.UserID != sess.UserID || loaded.HouseholdID != sess.HouseholdID {
		t.Fatalf("Load() = %#v, want %#v", loaded, sess)
	}

	if err := store.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := os.Stat(store.Path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("session file exists after delete: %v", err)
	}
}

func TestStoreDeleteMissing(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "missing.json"))
	if err := store.Delete(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Delete() error = %v, want os.ErrNotExist", err)
	}
}

func TestStoreSaveReturnsMkdirError(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := NewStore(filepath.Join(blocker, "session.json"))
	if err := store.Save(&Session{Profile: "default"}); err == nil {
		t.Fatal("Save() error = nil, want failure")
	}
}

func TestStoreLoadReturnsDecodeError(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "session.json"))
	if err := os.WriteFile(store.Path, []byte("not-json"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := store.Load(); err == nil {
		t.Fatal("Load() error = nil, want failure")
	}
}

func TestStoreLoadReturnsMissingFileError(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "missing.json"))
	if _, err := store.Load(); err == nil {
		t.Fatal("Load() error = nil, want failure")
	}
}

func TestStoreSaveReturnsWriteError(t *testing.T) {
	original := writeSessionFile
	writeSessionFile = func(string, []byte, os.FileMode) error {
		return errors.New("write failed")
	}
	defer func() { writeSessionFile = original }()

	store := NewStore(filepath.Join(t.TempDir(), "session.json"))
	if err := store.Save(&Session{Profile: "default"}); err == nil {
		t.Fatal("Save() error = nil, want failure")
	}
}

func TestStoreSaveReturnsMarshalError(t *testing.T) {
	original := marshalSession
	marshalSession = func(any, string, string) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}
	defer func() { marshalSession = original }()

	store := NewStore(filepath.Join(t.TempDir(), "session.json"))
	if err := store.Save(&Session{Profile: "default"}); err == nil {
		t.Fatal("Save() error = nil, want failure")
	}
}

func TestAuthenticate(t *testing.T) {
	originalEndpoint := loginEndpoint
	originalClientFactory := newLoginHTTPClient
	defer func() {
		loginEndpoint = originalEndpoint
		newLoginHTTPClient = originalClientFactory
	}()

	t.Run("invalid mfa secret", func(t *testing.T) {
		_, err := Authenticate("a@example.com", "password", "", "not-base32")
		assert.ErrorContains(t, err, "failed to generate MFA code")
	})

	t.Run("request creation error", func(t *testing.T) {
		loginEndpoint = "://"
		_, err := Authenticate("a@example.com", "password", "", "")
		assert.ErrorContains(t, err, "failed to create login request")
	})

	t.Run("network unreachable", func(t *testing.T) {
		loginEndpoint = "https://api.monarch.com/auth/login/"
		newLoginHTTPClient = func() *http.Client {
			return &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("network down")
			})}
		}
		_, err := Authenticate("a@example.com", "password", "", "")
		assert.ErrorContains(t, err, "failed to reach Monarch API")
	})

	t.Run("mfa required", func(t *testing.T) {
		newLoginHTTPClient = func() *http.Client {
			return &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 401, Body: io.NopCloser(bytes.NewBufferString(""))}, nil
			})}
		}
		_, err := Authenticate("a@example.com", "password", "", "")
		assert.ErrorContains(t, err, "MFA code required")
	})

	t.Run("invalid credentials with mfa", func(t *testing.T) {
		newLoginHTTPClient = func() *http.Client {
			return &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 401, Body: io.NopCloser(bytes.NewBufferString(""))}, nil
			})}
		}
		_, err := Authenticate("a@example.com", "password", "123456", "")
		assert.ErrorContains(t, err, "invalid credentials or MFA code")
	})

	t.Run("api error", func(t *testing.T) {
		newLoginHTTPClient = func() *http.Client {
			return &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewBufferString(""))}, nil
			})}
		}
		_, err := Authenticate("a@example.com", "password", "123456", "")
		assert.ErrorContains(t, err, "API returned status 500")
	})

	t.Run("schema changed", func(t *testing.T) {
		newLoginHTTPClient = func() *http.Client {
			return &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString("not-json"))}, nil
			})}
		}
		_, err := Authenticate("a@example.com", "password", "123456", "")
		assert.ErrorContains(t, err, "failed to parse login response")
	})

	t.Run("success", func(t *testing.T) {
		newLoginHTTPClient = func() *http.Client {
			return &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				body, _ := io.ReadAll(req.Body)
				require.Contains(t, string(body), `"username":"a@example.com"`)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"token":"token-123"}`))}, nil
			})}
		}
		sess, err := Authenticate("a@example.com", "password", "123456", "")
		require.NoError(t, err)
		require.NotNil(t, sess)
		assert.Equal(t, "token-123", sess.Token)
		assert.False(t, sess.CreatedAt.IsZero())
		assert.False(t, sess.UpdatedAt.IsZero())
	})

	t.Run("success with mfa secret", func(t *testing.T) {
		newLoginHTTPClient = func() *http.Client {
			return &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				body, _ := io.ReadAll(req.Body)
				require.Contains(t, string(body), `"totp"`)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"token":"token-456"}`))}, nil
			})}
		}
		sess, err := Authenticate("a@example.com", "password", "", "JBSWY3DPEHPK3PXP")
		require.NoError(t, err)
		require.NotNil(t, sess)
		assert.Equal(t, "token-456", sess.Token)
	})
}
