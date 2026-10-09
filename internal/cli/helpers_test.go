package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/auth"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = original }()
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, reader)
		done <- buf.String()
	}()
	fn()
	_ = writer.Close()
	out := <-done
	_ = reader.Close()
	return out
}

func saveTestSession(t *testing.T, sessionPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	s := auth.Session{
		Profile:   "default",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Token:     "test-token",
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error = %v", err)
	}
	if err := os.WriteFile(sessionPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func trimNewline(s string) string {
	for s != "" && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
