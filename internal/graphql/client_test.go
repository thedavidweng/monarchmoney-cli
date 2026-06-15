package graphql

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	clierrors "github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://example.invalid/graphql", "token", 3*time.Second)
	if client.Endpoint != "https://example.invalid/graphql" || client.Token != "token" || client.HTTP.Timeout != 3*time.Second {
		t.Fatalf("NewClient() returned %#v", client)
	}
}

func TestTokenValue(t *testing.T) {
	client := NewClient("https://example.invalid/graphql", "token", time.Second)
	if got := client.TokenValue(); got != "token" {
		t.Fatalf("TokenValue() = %q, want %q", got, "token")
	}
}

func TestDoSuccessAndHeaders(t *testing.T) {
	var gotReq *http.Request
	client := NewClient("https://example.invalid/graphql", "abc123", time.Second)
	client.HTTP = &http.Client{Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotReq = req
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"data":{"foo":"bar"}}`))}, nil
	})}

	var result struct {
		Foo string `json:"foo"`
	}
	err := client.Do(context.Background(), &Request{OperationName: "Test", Query: "query { foo }"}, &result)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if result.Foo != "bar" {
		t.Fatalf("result = %#v", result)
	}
	if got := gotReq.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := gotReq.Header.Get("Client-Platform"); got != "web" {
		t.Fatalf("Client-Platform = %q", got)
	}
	if got := gotReq.Header.Get("Authorization"); got != "Token abc123" {
		t.Fatalf("Authorization = %q", got)
	}
	if got := gotReq.Header.Get("User-Agent"); got != UserAgent() {
		t.Fatalf("User-Agent = %q, want %q", got, UserAgent())
	}
}

func TestDoWithoutTokenOmitsAuthorization(t *testing.T) {
	var gotReq *http.Request
	client := NewClient("https://example.invalid/graphql", "", time.Second)
	client.HTTP = &http.Client{Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotReq = req
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"data":{"foo":"bar"}}`))}, nil
	})}

	var result struct {
		Foo string `json:"foo"`
	}
	if err := client.Do(context.Background(), &Request{OperationName: "Test", Query: "query { foo }"}, &result); err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if got := gotReq.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want empty", got)
	}
}

func TestDoErrorPaths(t *testing.T) {
	t.Run("marshal request", func(t *testing.T) {
		client := NewClient("https://example.invalid/graphql", "", time.Second)
		err := client.Do(context.Background(), &Request{Variables: map[string]any{"bad": make(chan int)}}, &struct{}{})
		if err == nil {
			t.Fatal("Do() error = nil, want failure")
		}
	})

	t.Run("bad endpoint", func(t *testing.T) {
		client := NewClient("://", "", time.Second)
		err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
		if err == nil {
			t.Fatal("Do() error = nil, want failure")
		}
	})

	t.Run("network unreachable", func(t *testing.T) {
		client := NewClient("https://example.invalid/graphql", "", time.Second)
		client.HTTP = &http.Client{Transport: testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("boom")
		})}
		err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
		if err == nil {
			t.Fatal("Do() error = nil, want failure")
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		client := NewClient("https://example.invalid/graphql", "", time.Second)
		client.HTTP = &http.Client{Transport: testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 401, Body: io.NopCloser(bytes.NewBufferString("{}"))}, nil
		})}
		err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
		if err == nil {
			t.Fatal("Do() error = nil, want failure")
		}
		if got := err.Error(); !strings.Contains(got, "run `monarch auth login` again") {
			t.Fatalf("Do() error = %q, want re-login guidance", got)
		}
	})

	t.Run("non-200", func(t *testing.T) {
		client := NewClient("https://example.invalid/graphql", "", time.Second)
		client.HTTP = &http.Client{Transport: testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewBufferString("{}"))}, nil
		})}
		err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
		if err == nil {
			t.Fatal("Do() error = nil, want failure")
		}
	})

	t.Run("read body", func(t *testing.T) {
		client := NewClient("https://example.invalid/graphql", "", time.Second)
		client.HTTP = &http.Client{Transport: testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: testutil.FailingCloser{}}, nil
		})}
		err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
		if err == nil {
			t.Fatal("Do() error = nil, want failure")
		}
	})

	t.Run("schema changed", func(t *testing.T) {
		client := NewClient("https://example.invalid/graphql", "", time.Second)
		client.HTTP = &http.Client{Transport: testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString("not-json"))}, nil
		})}
		err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
		if err == nil {
			t.Fatal("Do() error = nil, want failure")
		}
	})

	t.Run("graphql errors", func(t *testing.T) {
		client := NewClient("https://example.invalid/graphql", "", time.Second)
		client.HTTP = &http.Client{Transport: testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"data":{},"errors":[{"message":"bad"}]}`))}, nil
		})}
		err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
		if err == nil {
			t.Fatal("Do() error = nil, want failure")
		}
	})
}

func TestDoErrorTypeIsStructured(t *testing.T) {
	client := NewClient("https://example.invalid/graphql", "", time.Second)
	client.HTTP = &http.Client{Transport: testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})}
	err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
	var clierr *clierrors.Error
	if !errors.As(err, &clierr) {
		t.Fatalf("error type = %T, want *errors.Error", err)
	}
}

func TestDoRetriesOnRetryableErrors(t *testing.T) {
	attempts := 0
	client := NewClient("https://example.invalid/graphql", "", 2*time.Second)
	client.HTTP = &http.Client{Transport: testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		if attempts <= 2 {
			return nil, errors.New("transient network error")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"data":{"ok":true}}`))}, nil
	})}

	var result struct {
		OK bool `json:"ok"`
	}
	err := client.Do(context.Background(), &Request{Query: "query { ok }"}, &result)
	if err != nil {
		t.Fatalf("Do() error = %v, want success after retries", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestDoDoesNotRetryNonRetryableErrors(t *testing.T) {
	attempts := 0
	client := NewClient("https://example.invalid/graphql", "", time.Second)
	client.HTTP = &http.Client{Transport: testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"data":{},"errors":[{"message":"bad query"}]}`))}, nil
	})}

	err := client.Do(context.Background(), &Request{Query: "bad"}, &struct{}{})
	if err == nil {
		t.Fatal("Do() error = nil, want failure")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1 (no retry for GraphQL errors)", attempts)
	}
}

func TestDoResponseSizeLimit(t *testing.T) {
	// Create a response larger than the 50MB limit.
	bigBody := strings.Repeat("x", int(maxResponseBody)+1)
	client := NewClient("https://example.invalid/graphql", "", 5*time.Second)
	client.HTTP = &http.Client{Transport: testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(bigBody))}, nil
	})}

	err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
	// Should fail because the oversized response is not valid JSON.
	if err == nil {
		t.Fatal("Do() error = nil, want failure for oversized response")
	}
}

func TestDoJoinsMultipleGraphQLErrors(t *testing.T) {
	client := NewClient("https://example.invalid/graphql", "", time.Second)
	client.HTTP = &http.Client{Transport: testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"data":{},"errors":[{"message":"error one"},{"message":"error two"}]}`))}, nil
	})}

	err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
	if err == nil {
		t.Fatal("Do() error = nil, want failure")
	}
	if got := err.Error(); !strings.Contains(got, "error one") || !strings.Contains(got, "error two") {
		t.Fatalf("Do() error = %q, want both errors joined", got)
	}
}

func TestUserAgentDefault(t *testing.T) {
	got := UserAgent()
	if got != DefaultUserAgent {
		t.Fatalf("UserAgent() = %q, want %q", got, DefaultUserAgent)
	}
}

func TestUserAgentEnvOverride(t *testing.T) {
	t.Setenv("MONARCH_USER_AGENT", "CustomBot/1.0")
	got := UserAgent()
	if got != "CustomBot/1.0" {
		t.Fatalf("UserAgent() = %q, want %q", got, "CustomBot/1.0")
	}
}
