package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func withAnalyzeCommandTestDefaults(t *testing.T, sessionPath string) *int {
	t.Helper()

	oldExitFunc := exitFunc
	oldDefaultSessionPath := defaultSessionPath
	oldJSONMode := jsonMode
	oldPretty := pretty
	oldProfile := profile
	oldTransport := http.DefaultTransport

	exitCode := 0
	exitFunc = func(code int) {
		exitCode = code
	}
	defaultSessionPath = func() string { return sessionPath }
	jsonMode = true
	pretty = false
	profile = "default"

	analyzeAnomaliesCmd.SetContext(context.Background())
	analyzeSubscriptionsCmd.SetContext(context.Background())
	analyzeMerchantsCmd.SetContext(context.Background())
	analyzeBurnRateCmd.SetContext(context.Background())

	t.Cleanup(func() {
		exitFunc = oldExitFunc
		defaultSessionPath = oldDefaultSessionPath
		jsonMode = oldJSONMode
		pretty = oldPretty
		profile = oldProfile
		http.DefaultTransport = oldTransport
	})

	return &exitCode
}

func TestAnalyzeAnomaliesJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withAnalyzeCommandTestDefaults(t, sessionPath)
	saveTestSession(t, sessionPath)

	var sawHistoryStart bool
	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string                 `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetTransactionsList" {
			t.Fatalf("operation = %q, want GetTransactionsList", gqlReq.OperationName)
		}
		filters := gqlReq.Variables["filters"].(map[string]interface{})
		if filters["startDate"] == "2025-11-01" && filters["endDate"] == "2026-05-31" {
			sawHistoryStart = true
		}
		return testutil.JSONResponse(`{"data":{"allTransactions":{"results":[
			{"id":"h1","date":"2025-11-15","amount":-100,"merchant":{"name":"Cafe"},"category":{"name":"Dining"},"account":{"id":"acc"}},
			{"id":"h2","date":"2025-12-15","amount":-100,"merchant":{"name":"Cafe"},"category":{"name":"Dining"},"account":{"id":"acc"}},
			{"id":"h3","date":"2026-01-15","amount":-100,"merchant":{"name":"Cafe"},"category":{"name":"Dining"},"account":{"id":"acc"}},
			{"id":"h4","date":"2026-02-15","amount":-100,"merchant":{"name":"Cafe"},"category":{"name":"Dining"},"account":{"id":"acc"}},
			{"id":"h5","date":"2026-03-15","amount":-100,"merchant":{"name":"Cafe"},"category":{"name":"Dining"},"account":{"id":"acc"}},
			{"id":"h6","date":"2026-04-15","amount":-100,"merchant":{"name":"Cafe"},"category":{"name":"Dining"},"account":{"id":"acc"}},
			{"id":"c1","date":"2026-05-03","amount":-300,"merchant":{"name":"Restaurant"},"category":{"name":"Dining"},"account":{"id":"acc"}}
		],"totalCount":7}}}`), nil
	})

	_ = analyzeAnomaliesCmd.Flags().Set("month", "2026-05")
	_ = analyzeAnomaliesCmd.Flags().Set("history-months", "6")
	_ = analyzeAnomaliesCmd.Flags().Set("min-ratio", "1.5")
	_ = analyzeAnomaliesCmd.Flags().Set("min-amount", "100")
	out := captureStdout(t, func() {
		analyzeAnomaliesCmd.Run(analyzeAnomaliesCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !sawHistoryStart {
		t.Fatal("anomalies did not request full history/current transaction window")
	}
	if !strings.Contains(out, `"command":"analyze.anomalies"`) || !strings.Contains(out, `"largest_merchant":"Restaurant"`) {
		t.Fatalf("output = %q", out)
	}
}

func TestAnalyzeMerchantsRejectsUnsupportedCompare(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withAnalyzeCommandTestDefaults(t, sessionPath)
	saveTestSession(t, sessionPath)

	_ = analyzeMerchantsCmd.Flags().Set("compare", "quarter")
	out := captureStdout(t, func() {
		analyzeMerchantsCmd.Run(analyzeMerchantsCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "previous-month") {
		t.Fatalf("output = %q, want supported compare guidance", out)
	}
}

func TestAnalyzeBurnRateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withAnalyzeCommandTestDefaults(t, sessionPath)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetJointPlanningData" {
			t.Fatalf("operation = %q, want GetJointPlanningData", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"budgetData":{"monthlyAmountsByCategory":[{"category":{"id":"cat","name":"Dining"},"monthlyAmounts":[{"month":"2026-05","plannedCashFlowAmount":600,"actualAmount":670}]}]}}}`), nil
	})

	_ = analyzeBurnRateCmd.Flags().Set("month", "2026-05")
	out := captureStdout(t, func() {
		analyzeBurnRateCmd.Run(analyzeBurnRateCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"analyze.burn-rate"`) || !strings.Contains(out, `"status":"overspending"`) {
		t.Fatalf("output = %q", out)
	}
}

func TestAnalyzeSubscriptionsJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withAnalyzeCommandTestDefaults(t, sessionPath)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string                 `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_GetUpcomingRecurringTransactionItems" {
			t.Fatalf("operation = %q, want recurring items", gqlReq.OperationName)
		}
		if gqlReq.Variables["startDate"] == "" || gqlReq.Variables["endDate"] == "" {
			t.Fatalf("variables = %#v, want start/end dates", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"recurringTransactionItems":[{"stream":{"id":"netflix","frequency":"monthly","amount":15.49,"isApproximate":false,"merchant":{"name":"Netflix"}},"date":"2026-05-01","isPast":false,"transactionId":"","amount":15.49,"amountDiff":0,"category":{"id":"cat","name":"Entertainment"},"account":{"id":"acc","displayName":"Checking"}}]}}`), nil
	})

	out := captureStdout(t, func() {
		analyzeSubscriptionsCmd.Run(analyzeSubscriptionsCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"analyze.subscriptions"`) || !strings.Contains(out, `"annual":185.88`) {
		t.Fatalf("output = %q", out)
	}
}

func TestAnalyzeAnomaliesRequiresAuth(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "missing.json")
	exitCode := withAnalyzeCommandTestDefaults(t, sessionPath)

	out := captureStdout(t, func() {
		analyzeAnomaliesCmd.Run(analyzeAnomaliesCmd, nil)
	})

	if *exitCode != 3 {
		t.Fatalf("exitCode = %d, want auth failure; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, "AUTH_REQUIRED") {
		t.Fatalf("output = %q", out)
	}
}
