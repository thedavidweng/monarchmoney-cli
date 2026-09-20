package cli

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestMerchants(t *testing.T) {
	t.Run("list", testMerchantsList)
	t.Run("show", testMerchantsShow)
	t.Run("update", testMerchantsUpdate)
	t.Run("delete", testMerchantsDelete)
}

func testMerchantsList(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, merchantsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_ListMerchants" {
			t.Fatalf("operation = %q, want Common_ListMerchants", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"merchants":[{"id":"m-1","name":"Whole Foods","logoUrl":"https://example.com/logo.png","transactionCount":42,"createdAt":"2026-01-01T00:00:00Z","recurringTransactionStream":null}]}}`), nil
	})

	out := captureStdout(t, func() {
		merchantsListCmd.Run(merchantsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"merchants.list"`) || !strings.Contains(out, "Whole Foods") {
		t.Fatalf("output = %q", out)
	}
}

func testMerchantsShow(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, merchantsShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_GetEditMerchant" {
			t.Fatalf("operation = %q, want Common_GetEditMerchant", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"merchant":{"id":"m-1","name":"Whole Foods","logoUrl":"","transactionCount":42,"ruleCount":3,"canBeDeleted":true,"createdAt":"2026-01-01T00:00:00Z","recurringTransactionStream":{"id":"rs-1"}}}}`), nil
	})

	out := captureStdout(t, func() {
		merchantsShowCmd.Run(merchantsShowCmd, []string{"m-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"merchants.show"`) || !strings.Contains(out, "rs-1") {
		t.Fatalf("output = %q", out)
	}
}

func testMerchantsUpdate(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, merchantsUpdateCmd)
	saveTestSession(t, sessionPath)

	_ = merchantsUpdateCmd.Flags().Set("name", "Whole Foods Market")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_UpdateMerchant" {
			t.Fatalf("operation = %q, want Common_UpdateMerchant", gqlReq.OperationName)
		}
		input, _ := gqlReq.Variables["input"].(map[string]any)
		if input["merchantId"] != "m-1" || input["name"] != "Whole Foods Market" {
			t.Fatalf("update input = %v", input)
		}
		return testutil.JSONResponse(`{"data":{"updateMerchant":{"merchant":{"id":"m-1","name":"Whole Foods Market","transactionCount":42},"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		merchantsUpdateCmd.Run(merchantsUpdateCmd, []string{"m-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"merchants.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testMerchantsDelete(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, merchantsDeleteCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_DeleteMerchant" {
			t.Fatalf("operation = %q, want Common_DeleteMerchant", gqlReq.OperationName)
		}
		if gqlReq.Variables["merchantId"] != "m-1" {
			t.Fatalf("delete variables = %v", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"deleteMerchant":{"success":true}}}`), nil
	})

	out := captureStdout(t, func() {
		merchantsDeleteCmd.Run(merchantsDeleteCmd, []string{"m-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"merchants.delete"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func TestMerchantsHumanOutput(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath,
		merchantsListCmd, merchantsShowCmd, merchantsUpdateCmd, merchantsDeleteCmd)
	saveTestSession(t, sessionPath)

	oldJSON := jsonMode
	jsonMode = false
	t.Cleanup(func() { jsonMode = oldJSON })

	_ = merchantsUpdateCmd.Flags().Set("name", "Whole Foods Market")
	t.Cleanup(func() { _ = merchantsUpdateCmd.Flags().Set("name", "") })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_ListMerchants":
			return testutil.JSONResponse(`{"data":{"merchants":[{"id":"m-1","name":"Whole Foods","transactionCount":42}]}}`), nil
		case "Common_GetEditMerchant":
			return testutil.JSONResponse(`{"data":{"merchant":{"id":"m-1","name":"Whole Foods","transactionCount":42,"ruleCount":3}}}`), nil
		case "Common_UpdateMerchant":
			return testutil.JSONResponse(`{"data":{"updateMerchant":{"merchant":{"id":"m-1","name":"Whole Foods Market"},"errors":null}}}`), nil
		case "Common_DeleteMerchant":
			return testutil.JSONResponse(`{"data":{"deleteMerchant":{"success":true}}}`), nil
		default:
			t.Fatalf("operation = %q", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		merchantsListCmd.Run(merchantsListCmd, nil)
		merchantsShowCmd.Run(merchantsShowCmd, []string{"m-1"})
		merchantsUpdateCmd.Run(merchantsUpdateCmd, []string{"m-1"})
		merchantsDeleteCmd.Run(merchantsDeleteCmd, []string{"m-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	for _, want := range []string{"Whole Foods", "Successfully renamed merchant", "Successfully deleted merchant"} {
		if !strings.Contains(out, want) {
			t.Fatalf("human output missing %q in %q", want, out)
		}
	}
}
