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

func TestBudgets(t *testing.T) {
	t.Run("list", testBudgetsListJSON)
	t.Run("list_api_error", testBudgetsListAPIError)
	t.Run("list_invalid_month", testBudgetsListInvalidMonth)
	t.Run("export", testBudgetsExportJSON)
	t.Run("reset", testBudgetsResetJSON)
	t.Run("reset_missing_month", testBudgetsResetMissingMonth)
	t.Run("reset_invalid_month", testBudgetsResetInvalidMonth)
	t.Run("flexible_set", testBudgetsFlexibleSetJSON)
	t.Run("flex_rollover_set", testBudgetsFlexRolloverSetJSON)
}

func testBudgetsListJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, budgetsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetJointPlanningData" {
			t.Fatalf("operation = %q, want GetJointPlanningData", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"budgetData":{"monthlyAmountsByCategory":[{"category":{"id":"cat-1","name":"Dining"},"monthlyAmounts":[{"month":"2026-06","plannedCashFlowAmount":300,"actualAmount":150.50}]},{"category":{"id":"cat-2","name":"Groceries"},"monthlyAmounts":[{"month":"2026-06","plannedCashFlowAmount":500,"actualAmount":425}]}]}}}`), nil
	})

	_ = budgetsListCmd.Flags().Set("month", "2026-06")
	out := captureStdout(t, func() {
		budgetsListCmd.Run(budgetsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"budgets.list"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"category_name":"Dining"`) {
		t.Fatalf("output missing Dining category = %q", out)
	}
	if !strings.Contains(out, `"planned":300`) {
		t.Fatalf("output missing planned amount = %q", out)
	}
	if !strings.Contains(out, `"actual":150.5`) {
		t.Fatalf("output missing actual amount = %q", out)
	}
}

func testBudgetsListAPIError(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, budgetsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	})

	_ = budgetsListCmd.Flags().Set("month", "2026-06")
	out := captureStdout(t, func() {
		budgetsListCmd.Run(budgetsListCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want API failure; output=%q", out)
	}
	if !strings.Contains(out, `"API_ERROR"`) {
		t.Fatalf("output = %q, want API_ERROR", out)
	}
}

func testBudgetsListInvalidMonth(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, budgetsListCmd)
	saveTestSession(t, sessionPath)

	_ = budgetsListCmd.Flags().Set("month", "2026/06")
	out := captureStdout(t, func() {
		budgetsListCmd.Run(budgetsListCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "YYYY-MM") {
		t.Fatalf("output = %q, want month format guidance", out)
	}
}

func testBudgetsExportJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, budgetsExportCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		return testutil.JSONResponse(`{"data":{"budgetData":{"monthlyAmountsByCategory":[{"category":{"id":"cat-1","name":"Rent"},"monthlyAmounts":[{"month":"2026-06","plannedCashFlowAmount":1500,"actualAmount":1500}]}]}}}`), nil
	})

	_ = budgetsExportCmd.Flags().Set("month", "2026-06")
	out := captureStdout(t, func() {
		budgetsExportCmd.Run(budgetsExportCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"budgets.export"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"category_name":"Rent"`) {
		t.Fatalf("output missing Rent category = %q", out)
	}
}

func testBudgetsResetJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, budgetsResetCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "ResetBudget" {
			t.Fatalf("operation = %q, want ResetBudget", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"resetBudget":{"ok":true}}}`), nil
	})

	monthStr = ""
	_ = budgetsResetCmd.Flags().Set("month", "2026-06")
	out := captureStdout(t, func() {
		budgetsResetCmd.Run(budgetsResetCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"budgets.reset"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"status":"budget reset"`) {
		t.Fatalf("output missing status = %q", out)
	}
}

func testBudgetsResetMissingMonth(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, budgetsResetCmd)
	saveTestSession(t, sessionPath)

	monthStr = ""
	out := captureStdout(t, func() {
		budgetsResetCmd.Run(budgetsResetCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "--month is required") {
		t.Fatalf("output = %q, want month required message", out)
	}
}

func testBudgetsResetInvalidMonth(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, budgetsResetCmd)
	saveTestSession(t, sessionPath)

	monthStr = ""
	_ = budgetsResetCmd.Flags().Set("month", "bad")
	out := captureStdout(t, func() {
		budgetsResetCmd.Run(budgetsResetCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "YYYY-MM") {
		t.Fatalf("output = %q, want month format guidance", out)
	}
}

func testBudgetsFlexibleSetJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, budgetsFlexibleSetCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "UpdateFlexibleBudget" {
			t.Fatalf("operation = %q, want UpdateFlexibleBudget", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"updateOrCreateFlexBudgetItem":{"flexBudgetItem":{"month":6}}}}`), nil
	})

	monthStr = ""
	budgetAmount = 0
	_ = budgetsFlexibleSetCmd.Flags().Set("month", "2026-06")
	_ = budgetsFlexibleSetCmd.Flags().Set("amount", "750.50")
	out := captureStdout(t, func() {
		budgetsFlexibleSetCmd.Run(budgetsFlexibleSetCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"budgets.flexible.set"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testBudgetsFlexRolloverSetJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, budgetsFlexRolloverSetCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "UpdateFlexRolloverSettings" {
			t.Fatalf("operation = %q, want UpdateFlexRolloverSettings", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"updateBudgetSettings":{"budgetRolloverPeriod":{"id":"period-1"}}}}`), nil
	})

	monthStr = ""
	budgetAmount = 0
	_ = budgetsFlexRolloverSetCmd.Flags().Set("month", "2026-06-01")
	_ = budgetsFlexRolloverSetCmd.Flags().Set("amount", "1000")
	out := captureStdout(t, func() {
		budgetsFlexRolloverSetCmd.Run(budgetsFlexRolloverSetCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"budgets.flex-rollover.set"`) {
		t.Fatalf("output missing command = %q", out)
	}
}
