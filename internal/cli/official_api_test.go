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

func TestAccountsListJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, accountsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetAccounts" {
			t.Fatalf("operation = %q, want GetAccounts", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"accounts":[
			{"id":"acc-1","displayName":"Checking","type":{"name":"cash","display":"Cash"},"subtype":{"name":"checking","display":"Checking"},"displayBalance":1250.50,"currentBalance":1250.50,"updatedAt":"2026-05-09","isHidden":false,"isAsset":true,"mask":"1234","isManual":false},
			{"id":"acc-2","displayName":"Credit Card","type":{"name":"credit","display":"Credit"},"subtype":{"name":"credit_card","display":"Credit Card"},"displayBalance":-450.00,"currentBalance":-450.00,"updatedAt":"2026-05-08","isHidden":false,"isAsset":false,"mask":"5678","isManual":false},
			{"id":"acc-3","displayName":"Savings","type":{"name":"cash","display":"Cash"},"subtype":{"name":"savings","display":"Savings"},"displayBalance":0.00,"currentBalance":0.00,"updatedAt":"2026-05-07","isHidden":false,"isAsset":true,"mask":"9012","isManual":true}
		],"householdPreferences":{"id":"pref-1","accountGroupOrder":["acc-1","acc-2","acc-3"]}}}`), nil
	})

	out := captureStdout(t, func() {
		accountsListCmd.Run(accountsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.list"`) || !strings.Contains(out, `"display_name":"Checking"`) {
		t.Fatalf("output = %q", out)
	}
	if !strings.Contains(out, `"display_balance":-450`) {
		t.Fatalf("output missing negative balance = %q", out)
	}
	if !strings.Contains(out, `"display_balance":0`) {
		t.Fatalf("output missing zero balance = %q", out)
	}
}

func TestTransactionsListWithEdgeCases(t *testing.T) {
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
			t.Fatalf("operation = %q, want GetTransactionsList", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"allTransactions":{"results":[
			{"id":"tx-1","date":"2026-05-01","amount":0,"merchant":{"name":"Free Trial"},"category":{"name":"Entertainment"},"account":{"id":"acc-1"},"notes":""},
			{"id":"tx-2","date":"2026-05-02","amount":-12.34,"merchant":{"name":"Café & Bakery"},"category":{"name":"Dining"},"account":{"id":"acc-1"},"notes":"latte"},
			{"id":"tx-3","date":"2026-05-03","amount":5000,"merchant":{"name":"Payroll"},"category":{"name":"Income"},"account":{"id":"acc-1"},"notes":null},
			{"id":"tx-4","date":"2026-05-04","amount":-0.01,"merchant":{"name":"Test Merchant"},"category":{"name":"Misc"},"account":{"id":"acc-1"},"notes":"micro transaction"}
		],"totalCount":4}}}`), nil
	})

	out := captureStdout(t, func() {
		transactionsListCmd.Run(transactionsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.list"`) {
		t.Fatalf("output = %q", out)
	}
	if !strings.Contains(out, `"amount":0`) {
		t.Fatalf("output missing zero amount = %q", out)
	}
	if !strings.Contains(out, "Café") {
		t.Fatalf("output missing special characters = %q", out)
	}
}

func TestBudgetsShowJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, budgetsShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string                 `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetJointPlanningData" {
			t.Fatalf("operation = %q, want GetJointPlanningData", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"budgetData":{"monthlyAmountsByCategory":[{"category":{"id":"cat-dining","name":"Dining"},"monthlyAmounts":[{"month":"2026-05","plannedCashFlowAmount":300,"actualAmount":245.50}]}]}}}`), nil
	})

	_ = budgetsShowCmd.Flags().Set("month", "2026-05")
	out := captureStdout(t, func() {
		budgetsShowCmd.Run(budgetsShowCmd, []string{"cat-dining"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"budgets.show"`) || !strings.Contains(out, `"category_name":"Dining"`) {
		t.Fatalf("output = %q", out)
	}
}

func TestCashflowSummaryJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, cashflowSummaryCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetCashflowSummary" {
			t.Fatalf("operation = %q, want GetCashflowSummary", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"aggregates":[{"summary":{"sumIncome":8500,"sumExpense":6200,"savings":2300,"savingsRate":0.2706}}]}}`), nil
	})

	_ = cashflowSummaryCmd.Flags().Set("from", "2026-05-01")
	_ = cashflowSummaryCmd.Flags().Set("to", "2026-05-31")
	out := captureStdout(t, func() {
		cashflowSummaryCmd.Run(cashflowSummaryCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"cashflow.summary"`) || !strings.Contains(out, `"income":8500`) {
		t.Fatalf("output = %q", out)
	}
}

func TestTransactionsShowJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, transactionsShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string                 `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetTransaction" {
			t.Fatalf("operation = %q, want GetTransaction", gqlReq.OperationName)
		}
		if gqlReq.Variables["id"] != "tx-123" {
			t.Fatalf("variables = %#v, want id tx-123", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"getTransaction":{"id":"tx-123","date":"2026-05-15","amount":-42.50,"merchant":{"name":"Café & Co"},"category":{"name":"Dining"},"notes":"lunch with émojis 🍕","pending":false,"hideFromReports":false,"plaidName":"CAFE AND CO","isRecurring":false,"reviewStatus":"reviewed","needsReview":false,"isSplitTransaction":false,"createdAt":"2026-05-15T10:00:00Z","updatedAt":"2026-05-15T10:00:00Z","account":{"id":"acc-1","displayName":"Checking"},"tags":[{"id":"tag-1","name":"food","color":"#ff0000","order":1}]}}}`), nil
	})

	out := captureStdout(t, func() {
		transactionsShowCmd.Run(transactionsShowCmd, []string{"tx-123"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.show"`) || !strings.Contains(out, "Café") {
		t.Fatalf("output = %q", out)
	}
	if !strings.Contains(out, "émojis") {
		t.Fatalf("output missing special chars in notes = %q", out)
	}
}

func TestCategoriesListJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, categoriesListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetCategories" {
			t.Fatalf("operation = %q, want GetCategories", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"categories":[
			{"id":"cat-1","name":"Dining","order":1,"icon":"utensils","group":{"id":"grp-1","name":"Food & Drink","type":"expense"}},
			{"id":"cat-2","name":"Income","order":2,"icon":"dollar","group":{"id":"grp-2","name":"Income","type":"income"}}
		]}}`), nil
	})

	out := captureStdout(t, func() {
		categoriesListCmd.Run(categoriesListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.list"`) || !strings.Contains(out, `"name":"Dining"`) {
		t.Fatalf("output = %q", out)
	}
	if !strings.Contains(out, "Food") {
		t.Fatalf("output missing group name = %q", out)
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
