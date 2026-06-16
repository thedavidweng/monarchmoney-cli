package cli

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestRecurring(t *testing.T) {
	t.Run("list", testRecurringListJSON)
	t.Run("update", testRecurringUpdateJSON)
}

func testRecurringListJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, recurringListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_GetUpcomingRecurringTransactionItems" {
			t.Fatalf("operation = %q, want Web_GetUpcomingRecurringTransactionItems", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"recurringTransactionItems":[
			{"stream":{"id":"rec-1","frequency":"monthly","amount":15.99,"isApproximate":false,"merchant":{"id":"m-1","name":"Netflix","logoUrl":""}},"date":"2026-06-15","isPast":false,"transactionId":"","amount":15.99,"amountDiff":0,"category":{"id":"cat-1","name":"Entertainment"},"account":{"id":"acc-1","displayName":"Checking"}},
			{"stream":{"id":"rec-2","frequency":"weekly","amount":50,"isApproximate":false,"merchant":{"id":"m-2","name":"Gym","logoUrl":""}},"date":"2026-06-16","isPast":false,"transactionId":"","amount":50,"amountDiff":0,"category":{"id":"cat-2","name":"Health"},"account":{"id":"acc-1","displayName":"Checking"}}
		]}}`), nil
	})

	out := captureStdout(t, func() {
		recurringListCmd.Run(recurringListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"recurring.list"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"merchant":"Netflix"`) {
		t.Fatalf("output missing Netflix = %q", out)
	}
	if !strings.Contains(out, `"frequency":"monthly"`) {
		t.Fatalf("output missing frequency = %q", out)
	}
	if !strings.Contains(out, `"amount":15.99`) {
		t.Fatalf("output missing amount = %q", out)
	}
}

func testRecurringUpdateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, recurringUpdateCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "UpdateRecurringTransaction" {
			t.Fatalf("operation = %q, want UpdateRecurringTransaction", gqlReq.OperationName)
		}
		if gqlReq.Variables["id"] != "rec-1" {
			t.Fatalf("variables id = %v, want rec-1", gqlReq.Variables["id"])
		}
		if gqlReq.Variables["amount"] != float64(19.99) {
			t.Fatalf("variables amount = %v, want 19.99", gqlReq.Variables["amount"])
		}
		return testutil.JSONResponse(`{"data":{"updateRecurringTransaction":{"recurringTransaction":{"id":"rec-1","amount":19.99}}}}`), nil
	})

	recurringAmount = 0
	_ = recurringUpdateCmd.Flags().Set("amount", "19.99")
	out := captureStdout(t, func() {
		recurringUpdateCmd.Run(recurringUpdateCmd, []string{"rec-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"recurring.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, "rec-1") {
		t.Fatalf("output missing ID = %q", out)
	}
}
