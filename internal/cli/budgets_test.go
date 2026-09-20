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
	t.Run("settings", testBudgetsSettingsJSON)
	t.Run("set_group", testBudgetsSetGroupJSON)
	t.Run("create", testBudgetsCreateJSON)
	t.Run("clear", testBudgetsClearJSON)
	t.Run("reset_rollover", testBudgetsResetRolloverJSON)
	t.Run("flex_rollover_show", testBudgetsFlexRolloverShowJSON)
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
		if gqlReq.OperationName != "Common_GetJointPlanningData" {
			t.Fatalf("operation = %q, want Common_GetJointPlanningData", gqlReq.OperationName)
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
		if gqlReq.OperationName != "Common_ResetBudget" {
			t.Fatalf("operation = %q, want Common_ResetBudget", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"resetBudget":{"errors":null}}}`), nil
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
		if gqlReq.OperationName != "Common_UpdateFlexBudgetMutation" {
			t.Fatalf("operation = %q, want Common_UpdateFlexBudgetMutation", gqlReq.OperationName)
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

func testBudgetsSettingsJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, budgetsSettingsCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_BudgetSettings" {
			t.Fatalf("operation = %q, want Common_BudgetSettings", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"budgetSystem":"flex","budgetApplyToFutureMonthsDefault":true,"budgetStatus":{"hasBudget":true,"hasTransactions":true}}}`), nil
	})

	out := captureStdout(t, func() {
		budgetsSettingsCmd.Run(budgetsSettingsCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"budgets.settings"`) || !strings.Contains(out, `"system":"flex"`) {
		t.Fatalf("output = %q", out)
	}
}

func testBudgetsSetGroupJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, budgetsSetGroupCmd)
	saveTestSession(t, sessionPath)

	monthStr = ""
	_ = budgetsSetGroupCmd.Flags().Set("month", "2026-06")
	_ = budgetsSetGroupCmd.Flags().Set("amount", "900")
	t.Cleanup(func() {
		_ = budgetsSetGroupCmd.Flags().Set("month", "")
		_ = budgetsSetGroupCmd.Flags().Set("amount", "0")
	})

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_UpdateBudgetItem" {
			t.Fatalf("operation = %q, want Common_UpdateBudgetItem", gqlReq.OperationName)
		}
		input, _ := gqlReq.Variables["input"].(map[string]any)
		if input["categoryGroupId"] != "grp-1" || input["amount"] != float64(900) {
			t.Fatalf("set-group input = %v", input)
		}
		return testutil.JSONResponse(`{"data":{"updateOrCreateBudgetItem":{"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		budgetsSetGroupCmd.Run(budgetsSetGroupCmd, []string{"grp-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"budgets.set-group"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testBudgetsCreateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, budgetsCreateCmd)
	saveTestSession(t, sessionPath)

	monthStr = ""
	_ = budgetsCreateCmd.Flags().Set("month", "2026-06")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_CreateBudgetForHousehold" {
			t.Fatalf("operation = %q, want Common_CreateBudgetForHousehold", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"createBudget":{"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		budgetsCreateCmd.Run(budgetsCreateCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"budgets.create"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testBudgetsClearJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, budgetsClearCmd)
	saveTestSession(t, sessionPath)

	monthStr = ""
	_ = budgetsClearCmd.Flags().Set("month", "2026-06")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_ClearAllMutation" {
			t.Fatalf("operation = %q, want Web_ClearAllMutation", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"clearBudget":{"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		budgetsClearCmd.Run(budgetsClearCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"budgets.clear"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testBudgetsResetRolloverJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, budgetsResetRolloverCmd)
	saveTestSession(t, sessionPath)

	monthStr = ""
	budgetCategoryID = ""
	_ = budgetsResetRolloverCmd.Flags().Set("month", "2026-06")
	_ = budgetsResetRolloverCmd.Flags().Set("category-id", "cat-1")
	t.Cleanup(func() {
		_ = budgetsResetRolloverCmd.Flags().Set("month", "")
		_ = budgetsResetRolloverCmd.Flags().Set("category-id", "")
	})

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_ResetRolloverMutation" {
			t.Fatalf("operation = %q, want Web_ResetRolloverMutation", gqlReq.OperationName)
		}
		input, _ := gqlReq.Variables["input"].(map[string]any)
		if input["categoryId"] != "cat-1" || input["startMonth"] != "2026-06-01" {
			t.Fatalf("reset-rollover input = %v", input)
		}
		return testutil.JSONResponse(`{"data":{"resetBudgetRollover":{"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		budgetsResetRolloverCmd.Run(budgetsResetRolloverCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"budgets.reset-rollover"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testBudgetsFlexRolloverShowJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, budgetsFlexRolloverShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_GetFlexibleGroupRolloverSettings" {
			t.Fatalf("operation = %q, want Web_GetFlexibleGroupRolloverSettings", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"budgetSystem":"flex","flexExpenseRolloverPeriod":{"id":"r-1","startMonth":"2026-01-01","startingBalance":250}}}`), nil
	})

	out := captureStdout(t, func() {
		budgetsFlexRolloverShowCmd.Run(budgetsFlexRolloverShowCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"budgets.flex-rollover.show"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func TestBudgetsGapHumanOutput(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath,
		budgetsSettingsCmd, budgetsSetGroupCmd, budgetsCreateCmd, budgetsClearCmd,
		budgetsResetRolloverCmd, budgetsFlexRolloverShowCmd)
	saveTestSession(t, sessionPath)

	oldJSON := jsonMode
	jsonMode = false
	t.Cleanup(func() { jsonMode = oldJSON })

	_ = budgetsSetGroupCmd.Flags().Set("month", "2026-06")
	_ = budgetsSetGroupCmd.Flags().Set("amount", "900")
	_ = budgetsCreateCmd.Flags().Set("month", "2026-06")
	_ = budgetsClearCmd.Flags().Set("month", "2026-06")
	_ = budgetsResetRolloverCmd.Flags().Set("month", "2026-06")
	_ = budgetsResetRolloverCmd.Flags().Set("category-id", "cat-1")
	t.Cleanup(func() {
		monthStr = ""
		budgetCategoryID = ""
		_ = budgetsSetGroupCmd.Flags().Set("month", "")
		_ = budgetsSetGroupCmd.Flags().Set("amount", "0")
		_ = budgetsCreateCmd.Flags().Set("month", "")
		_ = budgetsClearCmd.Flags().Set("month", "")
		_ = budgetsResetRolloverCmd.Flags().Set("month", "")
		_ = budgetsResetRolloverCmd.Flags().Set("category-id", "")
	})

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_BudgetSettings":
			return testutil.JSONResponse(`{"data":{"budgetSystem":"flex","budgetStatus":{"hasBudget":true,"hasTransactions":true}}}`), nil
		case "Common_UpdateBudgetItem":
			return testutil.JSONResponse(`{"data":{"updateOrCreateBudgetItem":{"errors":null}}}`), nil
		case "Common_CreateBudgetForHousehold":
			return testutil.JSONResponse(`{"data":{"createBudget":{"errors":null}}}`), nil
		case "Web_ClearAllMutation":
			return testutil.JSONResponse(`{"data":{"clearBudget":{"errors":null}}}`), nil
		case "Web_ResetRolloverMutation":
			return testutil.JSONResponse(`{"data":{"resetBudgetRollover":{"errors":null}}}`), nil
		case "Web_GetFlexibleGroupRolloverSettings":
			return testutil.JSONResponse(`{"data":{"budgetSystem":"flex","flexExpenseRolloverPeriod":{"id":"r-1","startMonth":"2026-01-01","startingBalance":250}}}`), nil
		default:
			t.Fatalf("operation = %q", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		budgetsSettingsCmd.Run(budgetsSettingsCmd, nil)
		budgetsSetGroupCmd.Run(budgetsSetGroupCmd, []string{"grp-1"})
		budgetsCreateCmd.Run(budgetsCreateCmd, nil)
		budgetsClearCmd.Run(budgetsClearCmd, nil)
		budgetsResetRolloverCmd.Run(budgetsResetRolloverCmd, nil)
		budgetsFlexRolloverShowCmd.Run(budgetsFlexRolloverShowCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	for _, want := range []string{"System:", "group grp-1", "created budget", "cleared budget", "reset rollover", "Starting balance"} {
		if !strings.Contains(out, want) {
			t.Fatalf("human output missing %q in %q", want, out)
		}
	}
}
