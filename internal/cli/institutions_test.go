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

func TestInstitutions(t *testing.T) {
	t.Run("list", testInstitutionsListJSON)
	t.Run("list_dedup", testInstitutionsListDedup)
	t.Run("list_api_error", testInstitutionsListAPIError)
	t.Run("health", testInstitutionsHealthJSON)
}

func testInstitutionsListJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, institutionsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_GetInstitutionSettings" {
			t.Fatalf("operation = %q, want Web_GetInstitutionSettings", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"credentials":[
			{"id":"cred-1","updateRequired":false,"disconnectedFromDataProviderAt":null,"dataProvider":"plaid","institution":{"id":"inst-1","plaidInstitutionId":"ins_1","name":"Chase","status":"active"}},
			{"id":"cred-2","updateRequired":false,"disconnectedFromDataProviderAt":null,"dataProvider":"plaid","institution":{"id":"inst-2","plaidInstitutionId":"ins_2","name":"Wells Fargo","status":"active"}}
		]}}`), nil
	})

	out := captureStdout(t, func() {
		institutionsListCmd.Run(institutionsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"institutions.list"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"name":"Chase"`) {
		t.Fatalf("output missing Chase = %q", out)
	}
	if !strings.Contains(out, `"name":"Wells Fargo"`) {
		t.Fatalf("output missing Wells Fargo = %q", out)
	}
}

func testInstitutionsListDedup(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, institutionsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		// Two credentials for the same institution
		return testutil.JSONResponse(`{"data":{"credentials":[
			{"id":"cred-1","updateRequired":false,"disconnectedFromDataProviderAt":null,"dataProvider":"plaid","institution":{"id":"inst-1","plaidInstitutionId":"ins_1","name":"Chase","status":"active"}},
			{"id":"cred-2","updateRequired":false,"disconnectedFromDataProviderAt":null,"dataProvider":"plaid","institution":{"id":"inst-1","plaidInstitutionId":"ins_1","name":"Chase","status":"active"}}
		]}}`), nil
	})

	out := captureStdout(t, func() {
		institutionsListCmd.Run(institutionsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	// Should only appear once after dedup
	count := strings.Count(out, `"name":"Chase"`)
	if count != 1 {
		t.Fatalf("Chase appeared %d times, want 1 (dedup); output=%q", count, out)
	}
}

func testInstitutionsListAPIError(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, institutionsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	})

	out := captureStdout(t, func() {
		institutionsListCmd.Run(institutionsListCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want API failure; output=%q", out)
	}
	if !strings.Contains(out, `"API_ERROR"`) {
		t.Fatalf("output = %q, want API_ERROR", out)
	}
}

func testInstitutionsHealthJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, institutionsHealthCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetCredentialSyncHealth" {
			t.Fatalf("operation = %q, want GetCredentialSyncHealth", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"credentials":[
			{"id":"cred-1","updateRequired":true,"disconnectedFromDataProviderAt":"","syncDisabledAt":"","syncDisabledReason":"","dataProvider":"MX","displayLastUpdatedAt":"2020-01-01T00:00:00Z","institution":{"id":"inst-1","name":"PayPal","url":"https://paypal.com"},"accounts":[{"id":"acc-1","displayName":"PayPal"}]}
		]}}`), nil
	})

	out := captureStdout(t, func() {
		institutionsHealthCmd.Run(institutionsHealthCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"institutions.health"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"needs_attention_count":1`) {
		t.Fatalf("output missing attention count = %q", out)
	}
}

func TestInstitutionsHumanOutputGap(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, institutionsListCmd, institutionsHealthCmd)
	saveTestSession(t, sessionPath)

	oldJSON := jsonMode
	jsonMode = false
	t.Cleanup(func() { jsonMode = oldJSON })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Web_GetInstitutionSettings":
			return testutil.JSONResponse(`{"data":{"credentials":[
				{"id":"cred-1","updateRequired":false,"disconnectedFromDataProviderAt":null,"dataProvider":"plaid","institution":{"id":"inst-1","plaidInstitutionId":"ins_1","name":"Chase","status":"active"}}
			],"accounts":[]}}`), nil
		case "GetCredentialSyncHealth":
			return testutil.JSONResponse(`{"data":{"credentials":[
				{"id":"cred-1","updateRequired":true,"disconnectedFromDataProviderAt":"","syncDisabledAt":"","syncDisabledReason":"","dataProvider":"MX","displayLastUpdatedAt":"2020-01-01T00:00:00Z","institution":{"id":"inst-1","name":"PayPal","url":"https://paypal.com"},"accounts":[{"id":"acc-1","displayName":"PayPal"}]}
			]}}`), nil
		default:
			t.Fatalf("operation = %q", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		institutionsListCmd.Run(institutionsListCmd, nil)
		institutionsHealthCmd.Run(institutionsHealthCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	for _, want := range []string{"Chase", "Connections:", "PayPal", "needs attention"} {
		if !strings.Contains(out, want) {
			t.Fatalf("human output missing %q in %q", want, out)
		}
	}
}
