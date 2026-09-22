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

func TestGoals(t *testing.T) {
	t.Run("list", testGoalsListJSON)
	t.Run("list_api_error", testGoalsListAPIError)
	t.Run("budgets", testGoalsBudgetsJSON)
	t.Run("show", testGoalsShowJSON)
	t.Run("create", testGoalsCreateJSON)
	t.Run("update", testGoalsUpdateJSON)
	t.Run("delete", testGoalsDeleteJSON)
	t.Run("archive", testGoalsArchiveJSON)
	t.Run("restore", testGoalsRestoreJSON)
	t.Run("priorities", testGoalsPrioritiesJSON)
	t.Run("link_account", testGoalsLinkAccountJSON)
	t.Run("unlink_account", testGoalsUnlinkAccountJSON)
	t.Run("events_list", testGoalsEventsListJSON)
	t.Run("events_contribute", testGoalsEventsContributeJSON)
	t.Run("events_withdraw", testGoalsEventsWithdrawJSON)
	t.Run("events_update", testGoalsEventsUpdateJSON)
	t.Run("events_delete", testGoalsEventsDeleteJSON)
	t.Run("budget", testGoalsBudgetJSON)
	t.Run("budget_set", testGoalsBudgetSetJSON)
	t.Run("contributions", testGoalsContributionsJSON)
	t.Run("contributions_set", testGoalsContributionsSetJSON)
}

const testGoalFixture = `{"id":"goal-1","type":"custom","name":"Emergency","status":"active","progress":0.5,"currentBalance":5000,"targetDate":"2027-01-01","targetAmount":10000,"plannedMonthlyContribution":200,"currentMonthPlannedContributionAmount":200,"spendingTotal":0,"netContribution":5000,"estimatedMonthsUntilCompletion":25,"forecastedCompletionDate":"2028-01-01","isSinkingFund":false,"priority":1}`

const testGoalEventFixture = `{"id":"ge-1","date":"2026-05-01","amount":200,"type":"contribution","notes":"may","goal":{"id":"goal-1","name":"Emergency"},"account":{"id":"acc-1","displayName":"Checking"}}`

func testGoalsListJSON(t *testing.T) {
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
		if gqlReq.OperationName != "Common_SavingsGoals" {
			t.Fatalf("operation = %q, want Common_SavingsGoals", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"savingsGoals":[{"id":"goal-1","name":"Vacation","type":"savings_goal","status":"active","progress":0.75,"currentBalance":7500,"targetAmount":10000,"plannedMonthlyContribution":500,"isSinkingFund":false,"priority":1},{"id":"goal-2","name":"Emergency Fund","type":"savings_goal","status":"active","progress":0.50,"currentBalance":15000,"targetAmount":30000,"plannedMonthlyContribution":1000,"isSinkingFund":false,"priority":2}]}}`), nil
	})

	out := captureStdout(t, func() {
		goalsListCmd.Run(goalsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.list"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"Vacation"`) {
		t.Fatalf("output missing Vacation = %q", out)
	}
	if !strings.Contains(out, `"Emergency Fund"`) {
		t.Fatalf("output missing Emergency Fund = %q", out)
	}
	if !strings.Contains(out, `"current_balance":7500`) {
		t.Fatalf("output missing current_balance = %q", out)
	}
	if !strings.Contains(out, `"target_amount":10000`) {
		t.Fatalf("output missing target_amount = %q", out)
	}
}

func testGoalsListAPIError(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, goalsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	})

	out := captureStdout(t, func() {
		goalsListCmd.Run(goalsListCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want API failure; output=%q", out)
	}
	if !strings.Contains(out, `"API_ERROR"`) {
		t.Fatalf("output = %q, want API_ERROR", out)
	}
}

func testGoalsBudgetsJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, goalsBudgetsCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetSavingsGoals" {
			t.Fatalf("operation = %q, want GetSavingsGoals", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"savingsGoalMonthlyBudgetAmounts":[{"id":"sgb-1","savingsGoal":{"id":"goal-1","name":"Vacation","type":"savings_goal","status":"active"},"monthlyAmounts":[{"month":"2026-05","plannedAmount":500,"actualAmount":450,"remainingAmount":50}]}]}}`), nil
	})

	monthStr = ""
	_ = goalsBudgetsCmd.Flags().Set("month", "2026-05")
	out := captureStdout(t, func() {
		goalsBudgetsCmd.Run(goalsBudgetsCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.budgets"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"goal_name":"Vacation"`) {
		t.Fatalf("output missing goal name = %q", out)
	}
	if !strings.Contains(out, `"planned":500`) {
		t.Fatalf("output missing planned amount = %q", out)
	}
}

func testGoalsShowJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, goalsShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_SavingsGoal" {
			t.Fatalf("operation = %q, want Common_SavingsGoal", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"savingsGoal":` + testGoalFixture + `}}`), nil
	})

	out := captureStdout(t, func() {
		goalsShowCmd.Run(goalsShowCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.show"`) || !strings.Contains(out, "Emergency") {
		t.Fatalf("output = %q", out)
	}
}

func testGoalsCreateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsCreateCmd)
	saveTestSession(t, sessionPath)

	_ = goalsCreateCmd.Flags().Set("name", "Emergency")
	t.Cleanup(func() { _ = goalsCreateCmd.Flags().Set("name", "") })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_CreateSavingsGoals":
			return testutil.JSONResponse(`{"data":{"createSavingsGoals":{"savingsGoals":[{"id":"goal-1","type":"custom"}],"errors":null}}}`), nil
		case "Common_SavingsGoal":
			return testutil.JSONResponse(`{"data":{"savingsGoal":` + testGoalFixture + `}}`), nil
		default:
			t.Fatalf("operation = %q, want goal create ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		goalsCreateCmd.Run(goalsCreateCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.create"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testGoalsUpdateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsUpdateCmd)
	saveTestSession(t, sessionPath)

	_ = goalsUpdateCmd.Flags().Set("name", "Emergency Fund")
	t.Cleanup(func() { _ = goalsUpdateCmd.Flags().Set("name", "") })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_UpdateSavingsGoal" {
			t.Fatalf("operation = %q, want Common_UpdateSavingsGoal", gqlReq.OperationName)
		}
		input, _ := gqlReq.Variables["input"].(map[string]any)
		if input["id"] != "goal-1" || input["name"] != "Emergency Fund" {
			t.Fatalf("update input = %v", input)
		}
		return testutil.JSONResponse(`{"data":{"updateSavingsGoal":{"savingsGoal":` + testGoalFixture + `,"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		goalsUpdateCmd.Run(goalsUpdateCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testGoalsDeleteJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsDeleteCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_DeleteSavingsGoal" {
			t.Fatalf("operation = %q, want Common_DeleteSavingsGoal", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"deleteSavingsGoal":{"success":true,"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		goalsDeleteCmd.Run(goalsDeleteCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.delete"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testGoalsArchiveJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsArchiveCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_ArchiveSavingsGoal" {
			t.Fatalf("operation = %q, want Common_ArchiveSavingsGoal", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"archiveSavingsGoal":{"savingsGoal":` + testGoalFixture + `,"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		goalsArchiveCmd.Run(goalsArchiveCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.archive"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testGoalsRestoreJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsRestoreCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_UnarchiveSavingsGoal" {
			t.Fatalf("operation = %q, want Common_UnarchiveSavingsGoal", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"unarchiveSavingsGoal":{"savingsGoal":` + testGoalFixture + `,"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		goalsRestoreCmd.Run(goalsRestoreCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.restore"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testGoalsPrioritiesJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsPrioritiesCmd)
	saveTestSession(t, sessionPath)

	_ = goalsPrioritiesCmd.Flags().Set("id", "goal-1")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_UpdateSavingsGoalsPriorities" {
			t.Fatalf("operation = %q, want Common_UpdateSavingsGoalsPriorities", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"updateSavingsGoalsPriorities":{"goals":[{"id":"goal-1","priority":0}],"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		goalsPrioritiesCmd.Run(goalsPrioritiesCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.priorities"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testGoalsLinkAccountJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsLinkAccountCmd)
	saveTestSession(t, sessionPath)

	_ = goalsLinkAccountCmd.Flags().Set("account", "acc-1")
	t.Cleanup(func() { _ = goalsLinkAccountCmd.Flags().Set("account", "") })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_CreateSavingsGoalAccountInitialContributions":
			return testutil.JSONResponse(`{"data":{"createGoalAccountInitialContributions":{"userNotice":null,"errors":null}}}`), nil
		case "Common_SavingsGoal":
			return testutil.JSONResponse(`{"data":{"savingsGoal":` + testGoalFixture + `}}`), nil
		default:
			t.Fatalf("operation = %q, want goal link ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		goalsLinkAccountCmd.Run(goalsLinkAccountCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.link-account"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testGoalsUnlinkAccountJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsUnlinkAccountCmd)
	saveTestSession(t, sessionPath)

	_ = goalsUnlinkAccountCmd.Flags().Set("account", "acc-1")
	t.Cleanup(func() { _ = goalsUnlinkAccountCmd.Flags().Set("account", "") })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_CreateSavingsGoalAccountInitialContributions":
			return testutil.JSONResponse(`{"data":{"createGoalAccountInitialContributions":{"userNotice":null,"errors":null}}}`), nil
		case "Common_SavingsGoal":
			return testutil.JSONResponse(`{"data":{"savingsGoal":` + testGoalFixture + `}}`), nil
		default:
			t.Fatalf("operation = %q, want goal unlink ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		goalsUnlinkAccountCmd.Run(goalsUnlinkAccountCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.unlink-account"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testGoalsEventsListJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, goalsEventsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_SavingsGoalEvents" {
			t.Fatalf("operation = %q, want Common_SavingsGoalEvents", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"savingsGoal":{"id":"goal-1","goalEvents":[` + testGoalEventFixture + `]}}}`), nil
	})

	out := captureStdout(t, func() {
		goalsEventsListCmd.Run(goalsEventsListCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.events.list"`) || !strings.Contains(out, "ge-1") {
		t.Fatalf("output = %q", out)
	}
}

func testGoalsEventsContributeJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsEventsContributeCmd)
	saveTestSession(t, sessionPath)

	_ = goalsEventsContributeCmd.Flags().Set("account", "acc-1")
	_ = goalsEventsContributeCmd.Flags().Set("amount", "200")
	t.Cleanup(func() {
		_ = goalsEventsContributeCmd.Flags().Set("account", "")
		_ = goalsEventsContributeCmd.Flags().Set("amount", "0")
	})

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_ContributeToSavingsGoal":
			return testutil.JSONResponse(`{"data":{"createSavingsGoalContribution":{"userNotice":null,"goalEvent":{"id":"ge-1"}}}}`), nil
		case "Common_SavingsGoalEvents":
			return testutil.JSONResponse(`{"data":{"savingsGoal":{"id":"goal-1","goalEvents":[` + testGoalEventFixture + `]}}}`), nil
		default:
			t.Fatalf("operation = %q, want goal contribute ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		goalsEventsContributeCmd.Run(goalsEventsContributeCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.events.contribute"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testGoalsEventsWithdrawJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsEventsWithdrawCmd)
	saveTestSession(t, sessionPath)

	_ = goalsEventsWithdrawCmd.Flags().Set("account", "acc-1")
	_ = goalsEventsWithdrawCmd.Flags().Set("amount", "50")
	t.Cleanup(func() {
		_ = goalsEventsWithdrawCmd.Flags().Set("account", "")
		_ = goalsEventsWithdrawCmd.Flags().Set("amount", "0")
	})

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_WithdrawFromSavingsGoal":
			return testutil.JSONResponse(`{"data":{"createSavingsGoalWithdrawal":{"goalEvent":{"id":"ge-2"}}}}`), nil
		case "Common_SavingsGoalEvents":
			return testutil.JSONResponse(`{"data":{"savingsGoal":{"id":"goal-1","goalEvents":[{"id":"ge-2","date":"2026-05-02","amount":50,"type":"withdrawal","notes":"","goal":{"id":"goal-1","name":"Emergency"},"account":{"id":"acc-1","displayName":"Checking"}}]}}}`), nil
		default:
			t.Fatalf("operation = %q, want goal withdraw ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		goalsEventsWithdrawCmd.Run(goalsEventsWithdrawCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.events.withdraw"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testGoalsEventsUpdateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsEventsUpdateCmd)
	saveTestSession(t, sessionPath)

	_ = goalsEventsUpdateCmd.Flags().Set("notes", "june")
	t.Cleanup(func() { _ = goalsEventsUpdateCmd.Flags().Set("notes", "") })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_UpdateSavingsGoalEvent" {
			t.Fatalf("operation = %q, want Common_UpdateSavingsGoalEvent", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"updateGoalEvent":{"goalEvent":{"id":"ge-1","date":"2026-05-01","amount":200,"type":"contribution","notes":"june"}}}}`), nil
	})

	out := captureStdout(t, func() {
		goalsEventsUpdateCmd.Run(goalsEventsUpdateCmd, []string{"ge-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.events.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testGoalsEventsDeleteJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsEventsDeleteCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_DeleteSavingsGoalEvent" {
			t.Fatalf("operation = %q, want Common_DeleteSavingsGoalEvent", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"deleteGoalEvent":{"success":true}}}`), nil
	})

	out := captureStdout(t, func() {
		goalsEventsDeleteCmd.Run(goalsEventsDeleteCmd, []string{"ge-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.events.delete"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testGoalsBudgetJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, goalsBudgetCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_SavingsGoalBudgetAmounts" {
			t.Fatalf("operation = %q, want Common_SavingsGoalBudgetAmounts", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"savingsGoal":{"id":"goal-1","monthlyBudgetAmounts":[{"id":"mba-1","month":"2026-05-01","plannedAmount":200,"actualAmount":200,"remainingAmount":0}]}}}`), nil
	})

	out := captureStdout(t, func() {
		goalsBudgetCmd.Run(goalsBudgetCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.budget"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testGoalsBudgetSetJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsBudgetSetCmd)
	saveTestSession(t, sessionPath)

	monthStr = ""
	_ = goalsBudgetSetCmd.Flags().Set("month", "2026-06")
	_ = goalsBudgetSetCmd.Flags().Set("amount", "250")
	t.Cleanup(func() {
		_ = goalsBudgetSetCmd.Flags().Set("month", "")
		_ = goalsBudgetSetCmd.Flags().Set("amount", "0")
	})

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_SetSavingsGoalBudgetAmount" {
			t.Fatalf("operation = %q, want Common_SetSavingsGoalBudgetAmount", gqlReq.OperationName)
		}
		input, _ := gqlReq.Variables["input"].(map[string]any)
		if input["savingsGoalId"] != "goal-1" || input["month"] != "2026-06-01" || input["amount"] != float64(250) {
			t.Fatalf("budget set input = %v", input)
		}
		return testutil.JSONResponse(`{"data":{"setSavingsGoalBudgetAmount":{"success":true,"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		goalsBudgetSetCmd.Run(goalsBudgetSetCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.budget.set"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func TestGoalsHumanOutputGap(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath,
		goalsShowCmd, goalsCreateCmd, goalsUpdateCmd, goalsDeleteCmd, goalsArchiveCmd,
		goalsRestoreCmd, goalsPrioritiesCmd, goalsLinkAccountCmd, goalsUnlinkAccountCmd,
		goalsEventsListCmd, goalsEventsContributeCmd, goalsEventsWithdrawCmd,
		goalsEventsUpdateCmd, goalsEventsDeleteCmd, goalsBudgetCmd, goalsBudgetSetCmd)
	saveTestSession(t, sessionPath)

	oldJSON := jsonMode
	jsonMode = false
	t.Cleanup(func() { jsonMode = oldJSON })

	_ = goalsCreateCmd.Flags().Set("name", "Emergency")
	_ = goalsUpdateCmd.Flags().Set("name", "Emergency Fund")
	_ = goalsPrioritiesCmd.Flags().Set("id", "goal-1")
	_ = goalsLinkAccountCmd.Flags().Set("account", "acc-1")
	_ = goalsUnlinkAccountCmd.Flags().Set("account", "acc-1")
	_ = goalsEventsContributeCmd.Flags().Set("account", "acc-1")
	_ = goalsEventsContributeCmd.Flags().Set("amount", "200")
	_ = goalsEventsWithdrawCmd.Flags().Set("account", "acc-1")
	_ = goalsEventsWithdrawCmd.Flags().Set("amount", "50")
	_ = goalsEventsUpdateCmd.Flags().Set("notes", "june")
	_ = goalsBudgetSetCmd.Flags().Set("month", "2026-06")
	_ = goalsBudgetSetCmd.Flags().Set("amount", "250")
	t.Cleanup(func() {
		_ = goalsCreateCmd.Flags().Set("name", "")
		_ = goalsUpdateCmd.Flags().Set("name", "")
		_ = goalsLinkAccountCmd.Flags().Set("account", "")
		_ = goalsUnlinkAccountCmd.Flags().Set("account", "")
		_ = goalsEventsContributeCmd.Flags().Set("account", "")
		_ = goalsEventsContributeCmd.Flags().Set("amount", "0")
		_ = goalsEventsWithdrawCmd.Flags().Set("account", "")
		_ = goalsEventsWithdrawCmd.Flags().Set("amount", "0")
		_ = goalsEventsUpdateCmd.Flags().Set("notes", "")
		_ = goalsBudgetSetCmd.Flags().Set("month", "")
		_ = goalsBudgetSetCmd.Flags().Set("amount", "0")
	})

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_SavingsGoal":
			return testutil.JSONResponse(`{"data":{"savingsGoal":` + testGoalFixture + `}}`), nil
		case "Common_CreateSavingsGoals":
			return testutil.JSONResponse(`{"data":{"createSavingsGoals":{"savingsGoals":[{"id":"goal-1","type":"custom"}],"errors":null}}}`), nil
		case "Common_UpdateSavingsGoal":
			return testutil.JSONResponse(`{"data":{"updateSavingsGoal":{"savingsGoal":` + testGoalFixture + `,"errors":null}}}`), nil
		case "Common_DeleteSavingsGoal":
			return testutil.JSONResponse(`{"data":{"deleteSavingsGoal":{"success":true,"errors":null}}}`), nil
		case "Common_ArchiveSavingsGoal":
			return testutil.JSONResponse(`{"data":{"archiveSavingsGoal":{"savingsGoal":` + testGoalFixture + `,"errors":null}}}`), nil
		case "Common_UnarchiveSavingsGoal":
			return testutil.JSONResponse(`{"data":{"unarchiveSavingsGoal":{"savingsGoal":` + testGoalFixture + `,"errors":null}}}`), nil
		case "Common_UpdateSavingsGoalsPriorities":
			return testutil.JSONResponse(`{"data":{"updateSavingsGoalsPriorities":{"goals":[],"errors":null}}}`), nil
		case "Common_CreateSavingsGoalAccountInitialContributions":
			return testutil.JSONResponse(`{"data":{"createGoalAccountInitialContributions":{"errors":null}}}`), nil
		case "Common_SavingsGoalEvents":
			return testutil.JSONResponse(`{"data":{"savingsGoal":{"id":"goal-1","goalEvents":[` + testGoalEventFixture + `]}}}`), nil
		case "Common_ContributeToSavingsGoal":
			return testutil.JSONResponse(`{"data":{"createSavingsGoalContribution":{"goalEvent":{"id":"ge-1"}}}}`), nil
		case "Common_WithdrawFromSavingsGoal":
			return testutil.JSONResponse(`{"data":{"createSavingsGoalWithdrawal":{"goalEvent":{"id":"ge-1"}}}}`), nil
		case "Common_UpdateSavingsGoalEvent":
			return testutil.JSONResponse(`{"data":{"updateGoalEvent":{"goalEvent":{"id":"ge-1","notes":"june"}}}}`), nil
		case "Common_DeleteSavingsGoalEvent":
			return testutil.JSONResponse(`{"data":{"deleteGoalEvent":{"success":true}}}`), nil
		case "Common_SavingsGoalBudgetAmounts":
			return testutil.JSONResponse(`{"data":{"savingsGoal":{"id":"goal-1","monthlyBudgetAmounts":[{"id":"mba-1","month":"2026-05-01","plannedAmount":200,"actualAmount":100,"remainingAmount":100}]}}}`), nil
		case "Common_SetSavingsGoalBudgetAmount":
			return testutil.JSONResponse(`{"data":{"setSavingsGoalBudgetAmount":{"success":true,"errors":null}}}`), nil
		default:
			t.Fatalf("operation = %q", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		goalsShowCmd.Run(goalsShowCmd, []string{"goal-1"})
		goalsCreateCmd.Run(goalsCreateCmd, nil)
		goalsUpdateCmd.Run(goalsUpdateCmd, []string{"goal-1"})
		goalsDeleteCmd.Run(goalsDeleteCmd, []string{"goal-1"})
		goalsArchiveCmd.Run(goalsArchiveCmd, []string{"goal-1"})
		goalsRestoreCmd.Run(goalsRestoreCmd, []string{"goal-1"})
		goalsPrioritiesCmd.Run(goalsPrioritiesCmd, nil)
		goalsLinkAccountCmd.Run(goalsLinkAccountCmd, []string{"goal-1"})
		goalsUnlinkAccountCmd.Run(goalsUnlinkAccountCmd, []string{"goal-1"})
		goalsEventsListCmd.Run(goalsEventsListCmd, []string{"goal-1"})
		goalsEventsContributeCmd.Run(goalsEventsContributeCmd, []string{"goal-1"})
		goalsEventsWithdrawCmd.Run(goalsEventsWithdrawCmd, []string{"goal-1"})
		goalsEventsUpdateCmd.Run(goalsEventsUpdateCmd, []string{"ge-1"})
		goalsEventsDeleteCmd.Run(goalsEventsDeleteCmd, []string{"ge-1"})
		goalsBudgetCmd.Run(goalsBudgetCmd, []string{"goal-1"})
		goalsBudgetSetCmd.Run(goalsBudgetSetCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	for _, want := range []string{"Emergency", "created goal", "updated goal", "deleted goal", "archived goal", "restored goal", "priorities", "linked account", "unlinked account", "Contributed", "Withdrew", "updated goal event", "deleted goal event", "budget amount"} {
		if !strings.Contains(out, want) {
			t.Fatalf("human output missing %q in %q", want, out)
		}
	}
}

func testGoalsContributionsJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, goalsContributionsCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetSavingsGoalContributions" {
			t.Fatalf("operation = %q, want GetSavingsGoalContributions", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"savingsGoal":{"id":"goal-1","name":"Emergency","monthlyBudgetAmounts":[{"month":"2026-05-01","totalPlannedAmount":200,"totalActualAmount":100,"totalRemainingAmount":100,"accountBreakdown":[{"account":{"id":"acc-1","displayName":"Checking"},"plannedAmount":200,"actualAmount":100,"remainingAmount":100}]}]}}}`), nil
	})

	out := captureStdout(t, func() {
		goalsContributionsCmd.Run(goalsContributionsCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.contributions"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"total_planned":200`) {
		t.Fatalf("output missing planned = %q", out)
	}
}

func testGoalsContributionsSetJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, goalsContributionsSetCmd)
	saveTestSession(t, sessionPath)

	_ = goalsContributionsSetCmd.Flags().Set("account", "acc-1")
	_ = goalsContributionsSetCmd.Flags().Set("amount", "50")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_UpdateSavingsGoal" {
			t.Fatalf("operation = %q, want Common_UpdateSavingsGoal", gqlReq.OperationName)
		}
		input, _ := gqlReq.Variables["input"].(map[string]any)
		if input["id"] != "goal-1" {
			t.Fatalf("input = %#v, want goal-1", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"updateSavingsGoal":{"savingsGoal":` + testGoalFixture + `,"errors":[]}}}`), nil
	})

	out := captureStdout(t, func() {
		goalsContributionsSetCmd.Run(goalsContributionsSetCmd, []string{"goal-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"goals.contributions.set"`) {
		t.Fatalf("output missing command = %q", out)
	}
}
