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
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (failingBody) Close() error             { return nil }

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
	client.HTTP = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
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
	if got := gotReq.Header.Get("User-Agent"); got != UserAgent {
		t.Fatalf("User-Agent = %q, want %q", got, UserAgent)
	}
}

func TestDoWithoutTokenOmitsAuthorization(t *testing.T) {
	var gotReq *http.Request
	client := NewClient("https://example.invalid/graphql", "", time.Second)
	client.HTTP = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
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
		err := client.Do(context.Background(), &Request{Variables: map[string]interface{}{"bad": make(chan int)}}, &struct{}{})
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
		client.HTTP = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("boom")
		})}
		err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
		if err == nil {
			t.Fatal("Do() error = nil, want failure")
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		client := NewClient("https://example.invalid/graphql", "", time.Second)
		client.HTTP = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
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
		client.HTTP = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewBufferString("{}"))}, nil
		})}
		err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
		if err == nil {
			t.Fatal("Do() error = nil, want failure")
		}
	})

	t.Run("read body", func(t *testing.T) {
		client := NewClient("https://example.invalid/graphql", "", time.Second)
		client.HTTP = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: failingBody{}}, nil
		})}
		err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
		if err == nil {
			t.Fatal("Do() error = nil, want failure")
		}
	})

	t.Run("schema changed", func(t *testing.T) {
		client := NewClient("https://example.invalid/graphql", "", time.Second)
		client.HTTP = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString("not-json"))}, nil
		})}
		err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
		if err == nil {
			t.Fatal("Do() error = nil, want failure")
		}
	})

	t.Run("graphql errors", func(t *testing.T) {
		client := NewClient("https://example.invalid/graphql", "", time.Second)
		client.HTTP = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
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
	client.HTTP = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})}
	err := client.Do(context.Background(), &Request{Query: "query { foo }"}, &struct{}{})
	var clierr *clierrors.Error
	if !errors.As(err, &clierr) {
		t.Fatalf("error type = %T, want *errors.Error", err)
	}
}
