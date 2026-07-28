package cli

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestOverviewJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, overviewCmd)
	saveTestSession(t, sessionPath)

	var mu sync.Mutex
	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		mu.Lock()
		defer mu.Unlock()
		switch gqlReq.OperationName {
		case "GetAccounts":
			return testutil.JSONResponse(`{"data":{"accounts":[{"id":"a1","displayName":"Checking","type":{"name":"bank","display":"Bank"},"subtype":{"name":"checking","display":"Checking"},"displayBalance":1000,"currentBalance":1000,"updatedAt":"2026-05-01","isHidden":false,"isAsset":true,"mask":"1234","isManual":false,"includeInNetWorth":true,"includeBalanceInNetWorth":false}]}}`), nil
		case "GetCashflowSummary":
			return testutil.JSONResponse(`{"data":{"aggregates":[{"summary":{"sumIncome":5000,"sumExpense":3000,"savings":2000,"savingsRate":0.4}}]}}`), nil
		case "GetTransactionsList":
			return testutil.JSONResponse(`{"data":{"allTransactions":{"results":[{"id":"t1","date":"2026-07-01","amount":-50,"pending":false,"category":{"id":"c1","name":"Food","group":{"id":"g1","name":"Expenses","type":"expense"}},"merchant":{"name":"Grocery Store","id":"m1"},"account":{"id":"a1","displayName":"Checking","order":0,"type":{"group":"bank"}}}],"totalCount":42}}}`), nil
		default:
			t.Fatalf("unexpected operation %q", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		overviewCmd.Run(overviewCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"overview"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"net_worth":1000`) {
		t.Fatalf("output missing net_worth = %q", out)
	}
	if !strings.Contains(out, `"account_count":1`) {
		t.Fatalf("output missing account_count = %q", out)
	}
	if !strings.Contains(out, `"transaction_total":42`) {
		t.Fatalf("output missing transaction_total = %q", out)
	}
}

func TestOverviewHuman(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, overviewCmd)
	jsonMode = false
	t.Cleanup(func() { jsonMode = true })
	saveTestSession(t, sessionPath)

	var mu sync.Mutex
	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		_ = json.NewDecoder(req.Body).Decode(&gqlReq)
		mu.Lock()
		defer mu.Unlock()
		switch gqlReq.OperationName {
		case "GetAccounts":
			return testutil.JSONResponse(`{"data":{"accounts":[{"id":"a1","displayName":"Checking","type":{"name":"bank","display":"Bank"},"subtype":{"name":"checking","display":"Checking"},"displayBalance":1000,"currentBalance":1000,"updatedAt":"2026-05-01","isHidden":false,"isAsset":true,"mask":"1234","isManual":false,"includeInNetWorth":true,"includeBalanceInNetWorth":false}]}}`), nil
		case "GetCashflowSummary":
			return testutil.JSONResponse(`{"data":{"aggregates":[{"summary":{"sumIncome":5000,"sumExpense":3000,"savings":2000,"savingsRate":0.4}}]}}`), nil
		case "GetTransactionsList":
			return testutil.JSONResponse(`{"data":{"allTransactions":{"results":[{"id":"t1","date":"2026-07-01","amount":-50,"pending":false,"category":{"id":"c1","name":"Food","group":{"id":"g1","name":"Expenses","type":"expense"}},"merchant":{"name":"Grocery Store","id":"m1"},"account":{"id":"a1","displayName":"Checking","order":0,"type":{"group":"bank"}}}],"totalCount":42}}}`), nil
		default:
			return testutil.JSONResponse(`{"data":{}}`), nil
		}
	})

	out := captureStdout(t, func() {
		overviewCmd.Run(overviewCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, "Net Worth:") {
		t.Fatalf("output missing Net Worth = %q", out)
	}
	if !strings.Contains(out, "Grocery Store") {
		t.Fatalf("output missing merchant = %q", out)
	}
}
