package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func withReadCommandTestDefaults(t *testing.T, sessionPath string, cmds ...*cobra.Command) *int {
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
	for _, cmd := range cmds {
		cmd.SetContext(context.Background())
	}

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

func TestAccountsBalanceAtJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, accountsBalanceAtCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string                 `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_GetDisplayBalanceAtDate" {
			t.Fatalf("operation = %q, want balance at date", gqlReq.OperationName)
		}
		if gqlReq.Variables["date"] != "2026-05-10" {
			t.Fatalf("variables = %#v, want date", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"accounts":[{"id":"acc-1","displayName":"Checking","displayBalance":42.25,"type":{"name":"cash","group":"asset"}}]}}`), nil
	})

	_ = accountsBalanceAtCmd.Flags().Set("date", "2026-05-10")
	_ = accountsBalanceAtCmd.Flags().Set("account-id", "acc-1")
	out := captureStdout(t, func() {
		accountsBalanceAtCmd.Run(accountsBalanceAtCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.balance-at"`) || !strings.Contains(out, `"display_name":"Checking"`) {
		t.Fatalf("output = %q", out)
	}
}

func TestCashflowTrendsRejectsInvalidPeriod(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, cashflowTrendsCmd)
	saveTestSession(t, sessionPath)

	_ = cashflowTrendsCmd.Flags().Set("from", "2026-01-01")
	_ = cashflowTrendsCmd.Flags().Set("to", "2026-03-31")
	_ = cashflowTrendsCmd.Flags().Set("period", "week")
	out := captureStdout(t, func() {
		cashflowTrendsCmd.Run(cashflowTrendsCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "month, quarter, or year") {
		t.Fatalf("output = %q, want period guidance", out)
	}
}

func TestGoalsListJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, goalsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_GoalsV2" {
			t.Fatalf("operation = %q, want goals", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"goalsV2":[{"id":"goal-1","name":"Vacation"}]}}`), nil
	})

	out := captureStdout(t, func() {
		goalsListCmd.Run(goalsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.list"`) || !strings.Contains(out, `"Vacation"`) {
		t.Fatalf("output = %q", out)
	}
}

func TestInvestmentsPerformanceRequiresSecurityID(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, investmentsPerformanceCmd)
	saveTestSession(t, sessionPath)

	out := captureStdout(t, func() {
		investmentsPerformanceCmd.Run(investmentsPerformanceCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "--security-id is required") {
		t.Fatalf("output = %q, want security guidance", out)
	}
}

func TestTransactionsListPassesExtendedFilters(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, transactionsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string                 `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetTransactionsList" {
			t.Fatalf("operation = %q, want transactions", gqlReq.OperationName)
		}
		filters := gqlReq.Variables["filters"].(map[string]interface{})
		if filters["isPending"] != true || filters["hideFromReports"] != false {
			t.Fatalf("filters = %#v, want pending/hide-from-reports", filters)
		}
		goals, ok := filters["goals"].([]interface{})
		if !ok || len(goals) != 2 || goals[0] != "goal-1" || goals[1] != "goal-2" {
			t.Fatalf("filters goals = %#v, want goal ids", filters["goals"])
		}
		return testutil.JSONResponse(`{"data":{"allTransactions":{"results":[],"totalCount":0}}}`), nil
	})

	_ = transactionsListCmd.Flags().Set("pending", "true")
	_ = transactionsListCmd.Flags().Set("hide-from-reports", "false")
	_ = transactionsListCmd.Flags().Set("goal-id", "goal-1,goal-2")
	out := captureStdout(t, func() {
		transactionsListCmd.Run(transactionsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.list"`) {
		t.Fatalf("output = %q", out)
	}
}
