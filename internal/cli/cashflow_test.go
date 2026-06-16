package cli

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestCashflow(t *testing.T) {
	t.Run("summary", testCashflowSummaryWithDates)
	t.Run("categories", testCashflowCategoriesJSON)
	t.Run("merchants", testCashflowMerchantsJSON)
	t.Run("spending", testCashflowSpendingJSON)
	t.Run("list", testCashflowListJSON)
	t.Run("trends", testCashflowTrendsJSON)
	t.Run("trends_missing_from", testCashflowTrendsMissingFrom)
	t.Run("trends_invalid_from", testCashflowTrendsInvalidFrom)
	t.Run("trends_invalid_to", testCashflowTrendsInvalidTo)
	t.Run("trends_invalid_group_by", testCashflowTrendsInvalidGroupBy)
	t.Run("trends_invalid_period", testCashflowTrendsInvalidPeriod)
}

func testCashflowSummaryWithDates(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, cashflowSummaryCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetCashflowSummary" {
			t.Fatalf("operation = %q, want GetCashflowSummary", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"aggregates":[{"summary":{"sumIncome":8500,"sumExpense":6200,"savings":2300,"savingsRate":0.2706}}]}}`), nil
	})

	cfStartDate = ""
	cfEndDate = ""
	_ = cashflowSummaryCmd.Flags().Set("from", "2026-01-01")
	_ = cashflowSummaryCmd.Flags().Set("to", "2026-03-31")
	out := captureStdout(t, func() {
		cashflowSummaryCmd.Run(cashflowSummaryCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"cashflow.summary"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"income":8500`) {
		t.Fatalf("output missing income = %q", out)
	}
	if !strings.Contains(out, `"expense":6200`) {
		t.Fatalf("output missing expense = %q", out)
	}
	if !strings.Contains(out, `"savings_rate":0.2706`) {
		t.Fatalf("output missing savings rate = %q", out)
	}
}

func testCashflowCategoriesJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, cashflowCategoriesCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetCashflowCategories" {
			t.Fatalf("operation = %q, want GetCashflowCategories", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"aggregates":[{"groupBy":{"category":{"id":"cat-1","name":"Dining"}},"summary":{"sum":-450.50}},{"groupBy":{"category":{"id":"cat-2","name":"Groceries"}},"summary":{"sum":-320}}]}}`), nil
	})

	cfStartDate = ""
	cfEndDate = ""
	_ = cashflowCategoriesCmd.Flags().Set("from", "2026-01-01")
	_ = cashflowCategoriesCmd.Flags().Set("to", "2026-03-31")
	out := captureStdout(t, func() {
		cashflowCategoriesCmd.Run(cashflowCategoriesCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"cashflow.categories"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"name":"Dining"`) {
		t.Fatalf("output missing Dining = %q", out)
	}
}

func testCashflowMerchantsJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, cashflowMerchantsCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetCashflowMerchants" {
			t.Fatalf("operation = %q, want GetCashflowMerchants", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"aggregates":[{"groupBy":{"merchant":{"id":"m-1","name":"Amazon"}},"summary":{"sumIncome":0,"sumExpense":-120.50}}]}}`), nil
	})

	cfStartDate = ""
	cfEndDate = ""
	_ = cashflowMerchantsCmd.Flags().Set("from", "2026-01-01")
	_ = cashflowMerchantsCmd.Flags().Set("to", "2026-03-31")
	out := captureStdout(t, func() {
		cashflowMerchantsCmd.Run(cashflowMerchantsCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"cashflow.merchants"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"name":"Amazon"`) {
		t.Fatalf("output missing Amazon = %q", out)
	}
}

func testCashflowSpendingJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, cashflowSpendingCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetCashflowCategories" {
			t.Fatalf("operation = %q, want GetCashflowCategories", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"aggregates":[{"groupBy":{"category":{"id":"cat-1","name":"Income"}},"summary":{"sum":5000}},{"groupBy":{"category":{"id":"cat-2","name":"Dining"}},"summary":{"sum":-200}}]}}`), nil
	})

	cfStartDate = ""
	cfEndDate = ""
	_ = cashflowSpendingCmd.Flags().Set("from", "2026-01-01")
	_ = cashflowSpendingCmd.Flags().Set("to", "2026-03-31")
	out := captureStdout(t, func() {
		cashflowSpendingCmd.Run(cashflowSpendingCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"cashflow.spending"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"total_income":5000`) {
		t.Fatalf("output missing total income = %q", out)
	}
	if !strings.Contains(out, `"total_expenses":200`) {
		t.Fatalf("output missing total expenses = %q", out)
	}
	if !strings.Contains(out, `"net":4800`) {
		t.Fatalf("output missing net = %q", out)
	}
}

func testCashflowListJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, cashflowListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetTransactionsList" {
			t.Fatalf("operation = %q, want GetTransactionsList", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"allTransactions":{"results":[{"id":"tx-1","date":"2026-05-01","amount":5000,"merchant":{"name":"Payroll"},"category":{"name":"Income"},"account":{"id":"acc-1"},"notes":""},{"id":"tx-2","date":"2026-05-01","amount":-12.34,"merchant":{"name":"Cafe"},"category":{"name":"Dining"},"account":{"id":"acc-1"},"notes":""},{"id":"tx-3","date":"2026-05-02","amount":-50,"merchant":{"name":"Store"},"category":{"name":"Shopping"},"account":{"id":"acc-1"},"notes":""}],"totalCount":3}}}`), nil
	})

	cfStartDate = ""
	cfEndDate = ""
	_ = cashflowListCmd.Flags().Set("from", "2026-05-01")
	_ = cashflowListCmd.Flags().Set("to", "2026-05-02")
	out := captureStdout(t, func() {
		cashflowListCmd.Run(cashflowListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"cashflow.list"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testCashflowTrendsJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, cashflowTrendsCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		return testutil.JSONResponse(`{"data":{"aggregates":[{"groupBy":{"category":{"id":"cat-1"},"month":"2026-01"},"summary":{"sum":-500,"sumIncome":0,"sumExpense":500}},{"groupBy":{"category":{"id":"cat-2"},"month":"2026-02"},"summary":{"sum":-300,"sumIncome":0,"sumExpense":300}}]}}`), nil
	})

	cfStartDate = ""
	cfEndDate = ""
	_ = cashflowTrendsCmd.Flags().Set("from", "2026-01-01")
	_ = cashflowTrendsCmd.Flags().Set("to", "2026-03-31")
	_ = cashflowTrendsCmd.Flags().Set("group-by", "category")
	_ = cashflowTrendsCmd.Flags().Set("period", "month")
	out := captureStdout(t, func() {
		cashflowTrendsCmd.Run(cashflowTrendsCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"cashflow.trends"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"group_id":"cat-1"`) {
		t.Fatalf("output missing group_id = %q", out)
	}
	if !strings.Contains(out, `"sum_income":0`) {
		t.Fatalf("output missing sum_income = %q", out)
	}
}

func testCashflowTrendsMissingFrom(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, cashflowTrendsCmd)
	saveTestSession(t, sessionPath)

	cfStartDate = ""
	cfEndDate = ""
	out := captureStdout(t, func() {
		cashflowTrendsCmd.Run(cashflowTrendsCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "--from and --to are required") {
		t.Fatalf("output = %q, want from/to required message", out)
	}
}

func testCashflowTrendsInvalidFrom(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, cashflowTrendsCmd)
	saveTestSession(t, sessionPath)

	cfStartDate = ""
	cfEndDate = ""
	_ = cashflowTrendsCmd.Flags().Set("from", "01-01-2026")
	_ = cashflowTrendsCmd.Flags().Set("to", "2026-03-31")
	out := captureStdout(t, func() {
		cashflowTrendsCmd.Run(cashflowTrendsCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "YYYY-MM-DD") {
		t.Fatalf("output = %q, want date format guidance", out)
	}
}

func testCashflowTrendsInvalidTo(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, cashflowTrendsCmd)
	saveTestSession(t, sessionPath)

	cfStartDate = ""
	cfEndDate = ""
	_ = cashflowTrendsCmd.Flags().Set("from", "2026-01-01")
	_ = cashflowTrendsCmd.Flags().Set("to", "bad-date")
	out := captureStdout(t, func() {
		cashflowTrendsCmd.Run(cashflowTrendsCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "YYYY-MM-DD") {
		t.Fatalf("output = %q, want date format guidance", out)
	}
}

func testCashflowTrendsInvalidGroupBy(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, cashflowTrendsCmd)
	saveTestSession(t, sessionPath)

	cfStartDate = ""
	cfEndDate = ""
	_ = cashflowTrendsCmd.Flags().Set("from", "2026-01-01")
	_ = cashflowTrendsCmd.Flags().Set("to", "2026-03-31")
	_ = cashflowTrendsCmd.Flags().Set("group-by", "week")
	out := captureStdout(t, func() {
		cashflowTrendsCmd.Run(cashflowTrendsCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "group-by must be category or category-group") {
		t.Fatalf("output = %q, want group-by guidance", out)
	}
}

func testCashflowTrendsInvalidPeriod(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, cashflowTrendsCmd)
	saveTestSession(t, sessionPath)

	cfStartDate = ""
	cfEndDate = ""
	cfTrendGroupBy = ""
	cfTrendPeriod = ""
	_ = cashflowTrendsCmd.Flags().Set("from", "2026-01-01")
	_ = cashflowTrendsCmd.Flags().Set("to", "2026-03-31")
	_ = cashflowTrendsCmd.Flags().Set("group-by", "category")
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
