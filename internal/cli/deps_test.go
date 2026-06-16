package cli

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	clierrors "github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

func TestWrapError(t *testing.T) {
	t.Run("plain error", func(t *testing.T) {
		plain := errors.New("something broke")
		wrapped := wrapError(plain, "operation failed")
		if wrapped.Code != clierrors.APIError {
			t.Fatalf("wrapError() code = %q, want %q", wrapped.Code, clierrors.APIError)
		}
		if wrapped.Message != "operation failed" {
			t.Fatalf("wrapError() message = %q, want %q", wrapped.Message, "operation failed")
		}
		if wrapped.Err != plain {
			t.Fatalf("wrapError() err = %v, want %v", wrapped.Err, plain)
		}
	})

	t.Run("structured error passthrough", func(t *testing.T) {
		structured := clierrors.New(clierrors.NetworkUnreachable, "cannot reach API", clierrors.CatNetwork, true, nil)
		wrapped := wrapError(structured, "ignored message")
		if wrapped != structured {
			t.Fatalf("wrapError() did not return same *errors.Error; got %v", wrapped)
		}
		if wrapped.Code != clierrors.NetworkUnreachable {
			t.Fatalf("wrapError() code = %q, want %q", wrapped.Code, clierrors.NetworkUnreachable)
		}
	})

	t.Run("nil error", func(t *testing.T) {
		wrapped := wrapError(nil, "nil test")
		if wrapped.Code != clierrors.APIError {
			t.Fatalf("wrapError(nil) code = %q, want %q", wrapped.Code, clierrors.APIError)
		}
		if wrapped.Err != nil {
			t.Fatalf("wrapError(nil) err = %v, want nil", wrapped.Err)
		}
	})
}

func TestMutateSuccess(t *testing.T) {
	renderer := output.NewRenderer(nil, nil, false, false)
	deps := CommandDeps{
		Start:    time.Now(),
		Renderer: renderer,
	}

	result, err := deps.Mutate("test.command", "res-123", func() (any, error) {
		return map[string]string{"status": "ok"}, nil
	}, "should not appear")

	if err != nil {
		t.Fatalf("Mutate() error = %v", err)
	}
	m, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("Mutate() result type = %T", result)
	}
	if m["status"] != "ok" {
		t.Fatalf("Mutate() result = %v", m)
	}
}

func TestMutateFailure(t *testing.T) {
	oldExitFunc := exitFunc
	oldJSONMode := jsonMode
	oldProfile := profile
	exitFunc = func(int) {}
	jsonMode = true
	profile = "default"
	defer func() {
		exitFunc = oldExitFunc
		jsonMode = oldJSONMode
		profile = oldProfile
	}()

	renderer := output.NewRenderer(nil, nil, true, false)
	deps := CommandDeps{
		Start:    time.Now(),
		Renderer: renderer,
	}

	result, err := deps.Mutate("test.fail", "res-456", func() (any, error) {
		return nil, errors.New("boom")
	}, "operation failed")

	if err == nil {
		t.Fatal("Mutate() expected error, got nil")
	}
	if result != nil {
		t.Fatalf("Mutate() result = %v, want nil", result)
	}
}

func TestMutateStructuredError(t *testing.T) {
	oldExitFunc := exitFunc
	oldJSONMode := jsonMode
	oldProfile := profile
	exitFunc = func(int) {}
	jsonMode = true
	profile = "default"
	defer func() {
		exitFunc = oldExitFunc
		jsonMode = oldJSONMode
		profile = oldProfile
	}()

	renderer := output.NewRenderer(nil, nil, true, false)
	deps := CommandDeps{
		Start:    time.Now(),
		Renderer: renderer,
	}

	structuredErr := clierrors.New(clierrors.ValidationFailed, "bad input", clierrors.CatValidation, false, nil)
	result, err := deps.Mutate("test.structured", "res-789", func() (any, error) {
		return nil, structuredErr
	}, "validation error")

	if err != structuredErr {
		t.Fatalf("Mutate() err = %v, want structuredErr", err)
	}
	if result != nil {
		t.Fatalf("Mutate() result = %v, want nil", result)
	}
}

func TestMutateAuditLog(t *testing.T) {
	// Verify Mutate returns the result from fn without interacting with
	// a real audit log (the Logger.Log call is best-effort and may fail
	// silently if no audit dir exists).
	renderer := output.NewRenderer(nil, nil, false, false)
	deps := CommandDeps{
		Start:    time.Now(),
		Renderer: renderer,
	}

	type payload struct {
		ID string `json:"id"`
	}

	got, err := deps.Mutate("test.audit", "res-100", func() (any, error) {
		return &payload{ID: "abc"}, nil
	}, "should not appear")

	if err != nil {
		t.Fatalf("Mutate() error = %v", err)
	}
	p, ok := got.(*payload)
	if !ok {
		t.Fatalf("Mutate() result type = %T", got)
	}
	if p.ID != "abc" {
		t.Fatalf("Mutate() result.ID = %q", p.ID)
	}
}

func TestMutateJSONOutput(t *testing.T) {
	// Verify that when Mutate fails with jsonMode=true, the error envelope
	// is rendered to stdout as JSON.
	oldExitFunc := exitFunc
	oldJSONMode := jsonMode
	oldProfile := profile
	exitFunc = func(int) {}
	jsonMode = true
	profile = "test-profile"
	defer func() {
		exitFunc = oldExitFunc
		jsonMode = oldJSONMode
		profile = oldProfile
	}()

	out := captureStdout(t, func() {
		renderer := output.NewRenderer(nil, nil, true, false)
		deps := CommandDeps{
			Start:    time.Now(),
			Renderer: renderer,
		}

		deps.Mutate("test.json", "res-json", func() (any, error) {
			return nil, errors.New("fail")
		}, "operation failed")
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
	if env.OK {
		t.Fatal("expected ok=false in error envelope")
	}
}

func trimNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
