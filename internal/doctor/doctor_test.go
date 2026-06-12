//go:build !windows

package doctor

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/auth"
	"github.com/thedavidweng/monarchmoney-cli/internal/config"
	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestCheckWithoutLocalState(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	res := Check(context.Background(), false)

	if res.Version == "" || res.OS == "" || res.Arch == "" {
		t.Fatalf("Check() returned incomplete identity: %#v", res)
	}
	if res.Config.Exists || res.Session.Exists || res.Session.Authenticated || res.Network.APIReachable {
		t.Fatalf("Check() returned unexpected state: %#v", res)
	}
}

func TestCheckWithSessionAndConnectivity(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.MkdirAll(config.DefaultDir(), 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(config.DefaultConfigPath(), []byte("profile: default\n"), 0600); err != nil {
		t.Fatalf("WriteFile() config error = %v", err)
	}

	sess := &auth.Session{Profile: "default", Token: "token-123", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := auth.NewStore(config.DefaultSessionPath()).Save(sess); err != nil {
		t.Fatalf("Save() session error = %v", err)
	}
	if err := os.Chmod(config.DefaultSessionPath(), 0644); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	originalTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = originalTransport }()
	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		if !bytes.Contains(body, []byte("GetIdentity")) {
			t.Fatalf("unexpected GraphQL request body: %s", string(body))
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"data":{"identity":{"id":"me"}}}`))}, nil
	})

	res := Check(context.Background(), true)
	if !res.Config.Exists || !res.Session.Exists || !res.Session.Authenticated || res.Session.PermissionOK || !res.Network.APIReachable {
		t.Fatalf("Check() returned unexpected state: %#v", res)
	}
}
