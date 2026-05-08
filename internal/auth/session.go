package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Cookie represents a simple browser cookie.
type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Expires  int64  `json:"expires"`
	Secure   bool   `json:"secure"`
	HTTPOnly bool   `json:"http_only"`
}

// Session represents a Monarch Money authenticated session.
type Session struct {
	Profile     string     `json:"profile"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Token       string     `json:"token,omitempty"`
	Cookies     []Cookie   `json:"cookies,omitempty"`
	UserID      string     `json:"user_id,omitempty"`
	HouseholdID string     `json:"household_id,omitempty"`
}

// Store handles session persistence.
type Store struct {
	Path string
}

// NewStore returns a new Store.
func NewStore(path string) *Store {
	return &Store{Path: path}
}

// Save saves the session to disk with restricted permissions.
func (s *Store) Save(sess *Session) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.Path, data, 0600)
}

// Load loads the session from disk.
func (s *Store) Load() (*Session, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return nil, err
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}

	return &sess, nil
}

// Delete removes the session file.
func (s *Store) Delete() error {
	return os.Remove(s.Path)
}
