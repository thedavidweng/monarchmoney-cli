package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestSubscription(t *testing.T) {
	t.Run("show", testSubscriptionShowJSON)
	t.Run("show_api_error", testSubscriptionShowAPIError)
}

func testSubscriptionShowJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, subscriptionShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetSubscriptionDetails" {
			t.Fatalf("operation = %q, want GetSubscriptionDetails", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"subscription":{"id":"sub-1","paymentSource":"Visa ending 1234","referralCode":"ABC123","isOnFreeTrial":false,"hasPremiumEntitlement":true}}}`), nil
	})

	out := captureStdout(t, func() {
		subscriptionShowCmd.Run(subscriptionShowCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"subscription.show"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"payment_source":"Visa ending 1234"`) {
		t.Fatalf("output missing payment source = %q", out)
	}
	if !strings.Contains(out, `"referral_code":"ABC123"`) {
		t.Fatalf("output missing referral code = %q", out)
	}
	if !strings.Contains(out, `"has_premium_entitlement":true`) {
		t.Fatalf("output missing premium entitlement = %q", out)
	}
	// Verify the legacy GraphQL warning is present
	if !strings.Contains(out, "uses legacy Monarch GraphQL root field") {
		t.Fatalf("output missing legacy warning = %q", out)
	}
}

func testSubscriptionShowAPIError(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, subscriptionShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	})

	out := captureStdout(t, func() {
		subscriptionShowCmd.Run(subscriptionShowCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want API failure; output=%q", out)
	}
	if !strings.Contains(out, `"API_ERROR"`) {
		t.Fatalf("output = %q, want API_ERROR", out)
	}
}
