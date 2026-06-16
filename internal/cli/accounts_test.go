package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestAccounts(t *testing.T) {
	t.Run("list", testAccountsListJSON)
	t.Run("list_api_error", testAccountsListAPIError)
	t.Run("show", testAccountsShowJSON)
	t.Run("types", testAccountsTypesJSON)
	t.Run("holdings", testAccountsHoldingsJSON)
	t.Run("holdings_missing_arg", testAccountsHoldingsMissingArg)
	t.Run("balance_at", testAccountsBalanceAtJSON)
	t.Run("history", testAccountsHistoryJSON)
	t.Run("refresh", testAccountsRefreshJSON)
	t.Run("refresh_status", testAccountsRefreshStatusJSON)
	t.Run("update", testAccountsUpdateJSON)
	t.Run("delete", testAccountsDeleteJSON)
	t.Run("create_manual", testAccountsCreateManualJSON)
	t.Run("upload_history", testAccountsUploadHistoryJSON)
	t.Run("recent_balances", testAccountsRecentBalancesJSON)
	t.Run("snapshots", testAccountsSnapshotsJSON)
	t.Run("aggregate_snapshots", testAccountsAggregateSnapshotsJSON)
	t.Run("networth", testNetworthJSON)
}

func testAccountsListJSON(t *testing.T) {
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
		return testutil.JSONResponse(`{"data":{"accounts":[{"id":"a1","displayName":"Checking","type":{"name":"bank","display":"Bank"},"subtype":{"name":"checking","display":"Checking"},"displayBalance":42.5,"currentBalance":42.5,"updatedAt":"2026-05-01","isHidden":false,"isAsset":true,"mask":"1234","isManual":false}]}}`), nil
	})

	out := captureStdout(t, func() {
		accountsListCmd.Run(accountsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.list"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"display_name":"Checking"`) {
		t.Fatalf("output missing display name = %q", out)
	}
	if !strings.Contains(out, `"display_balance":42.5`) {
		t.Fatalf("output missing balance = %q", out)
	}
}

func testAccountsListAPIError(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, accountsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	})

	out := captureStdout(t, func() {
		accountsListCmd.Run(accountsListCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want API failure; output=%q", out)
	}
	if !strings.Contains(out, `"API_ERROR"`) {
		t.Fatalf("output = %q, want API_ERROR", out)
	}
}

func testAccountsShowJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, accountsShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetAccount" {
			t.Fatalf("operation = %q, want GetAccount", gqlReq.OperationName)
		}
		if gqlReq.Variables["id"] != "acc-1" {
			t.Fatalf("variables = %#v, want id=acc-1", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"account":{"id":"acc-1","displayName":"Cash","type":{"name":"cash","display":"Cash"},"subtype":{"name":"cash","display":"Cash"},"displayBalance":9.5,"currentBalance":9.5,"updatedAt":"2026-05-01","isHidden":false,"isAsset":true,"mask":"","isManual":true}}}`), nil
	})

	out := captureStdout(t, func() {
		accountsShowCmd.Run(accountsShowCmd, []string{"acc-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.show"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"display_name":"Cash"`) {
		t.Fatalf("output missing display name = %q", out)
	}
	if !strings.Contains(out, `"display_balance":9.5`) {
		t.Fatalf("output missing balance = %q", out)
	}
}

func testAccountsTypesJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, accountsTypesCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetAccountTypeOptions" {
			t.Fatalf("operation = %q, want GetAccountTypeOptions", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"accountTypes":[{"name":"bank","display":"Bank"},{"name":"credit","display":"Credit"}]}}`), nil
	})

	out := captureStdout(t, func() {
		accountsTypesCmd.Run(accountsTypesCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.types"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, "bank") {
		t.Fatalf("output missing bank type = %q", out)
	}
	if !strings.Contains(out, "credit") {
		t.Fatalf("output missing credit type = %q", out)
	}
}

func testAccountsHoldingsJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, accountsHoldingsCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_GetHoldings" {
			t.Fatalf("operation = %q, want Web_GetHoldings", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"portfolio":{"aggregateHoldings":{"edges":[{"node":{"id":"h1","quantity":2,"basis":3,"totalValue":6,"holdings":[{"id":"sub-1","quantity":2,"name":"VTI","ticker":"VTI","account":{"id":"acc-1"}}]}}]}}}}`), nil
	})

	out := captureStdout(t, func() {
		accountsHoldingsCmd.Run(accountsHoldingsCmd, []string{"acc-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.holdings"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"id":"h1"`) {
		t.Fatalf("output missing holding id = %q", out)
	}
	if !strings.Contains(out, `"total_value":6`) {
		t.Fatalf("output missing total value = %q", out)
	}
}

func testAccountsHoldingsMissingArg(t *testing.T) {
	// cobra.ExactArgs(1) is enforced by Execute(), not Run().
	// Calling Run directly with nil args would panic on args[0].
	// Verify the command declares the arg requirement.
	if accountsHoldingsCmd.Args == nil {
		t.Fatal("accountsHoldingsCmd.Args is nil, want cobra.ExactArgs(1)")
	}
	argsErr := accountsHoldingsCmd.Args(accountsHoldingsCmd, nil)
	if argsErr == nil {
		t.Fatal("Expected arg validation error for nil args, got nil")
	}
}

func testAccountsBalanceAtJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, accountsBalanceAtCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_GetDisplayBalanceAtDate" {
			t.Fatalf("operation = %q, want Common_GetDisplayBalanceAtDate", gqlReq.OperationName)
		}
		if gqlReq.Variables["date"] != "2026-05-10" {
			t.Fatalf("variables = %#v, want date=2026-05-10", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"accounts":[{"id":"a1","displayName":"Checking","displayBalance":42.25,"type":{"name":"cash","group":"asset"}}]}}`), nil
	})

	balanceAtDate = ""
	_ = accountsBalanceAtCmd.Flags().Set("date", "2026-05-10")
	out := captureStdout(t, func() {
		accountsBalanceAtCmd.Run(accountsBalanceAtCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.balance-at"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"display_name":"Checking"`) {
		t.Fatalf("output missing display name = %q", out)
	}
	if !strings.Contains(out, `"display_balance":42.25`) {
		t.Fatalf("output missing balance = %q", out)
	}
}

func testAccountsHistoryJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, accountsHistoryCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetAccountHistory" {
			t.Fatalf("operation = %q, want GetAccountHistory", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"aggregateSnapshots":[{"date":"2026-05-01","balance":10},{"date":"2026-05-02","balance":20}]}}`), nil
	})

	historyFrom = ""
	historyTo = ""
	_ = accountsHistoryCmd.Flags().Set("from", "2026-05-01")
	_ = accountsHistoryCmd.Flags().Set("to", "2026-05-31")
	out := captureStdout(t, func() {
		accountsHistoryCmd.Run(accountsHistoryCmd, []string{"acc-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.history"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"date":"2026-05-01"`) {
		t.Fatalf("output missing date = %q", out)
	}
	if !strings.Contains(out, `"amount":10`) {
		t.Fatalf("output missing amount = %q", out)
	}
}

func testAccountsRefreshJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, accountsRefreshCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "RefreshAccounts" {
			t.Fatalf("operation = %q, want RefreshAccounts", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"requestAccountsRefresh":{"ok":true}}}`), nil
	})

	out := captureStdout(t, func() {
		accountsRefreshCmd.Run(accountsRefreshCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.refresh"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"status":"refresh requested"`) {
		t.Fatalf("output missing status = %q", out)
	}
}

func testAccountsRefreshStatusJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, accountsRefreshStatusCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetAccountsRefreshStatus" {
			t.Fatalf("operation = %q, want GetAccountsRefreshStatus", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"accounts":[{"id":"a1","hasSyncInProgress":false}]}}`), nil
	})

	out := captureStdout(t, func() {
		accountsRefreshStatusCmd.Run(accountsRefreshStatusCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.refresh-status"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"is_complete":true`) {
		t.Fatalf("output missing is_complete = %q", out)
	}
}

func testAccountsUpdateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, accountsUpdateCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "UpdateAccount" {
			t.Fatalf("operation = %q, want UpdateAccount", gqlReq.OperationName)
		}
		if gqlReq.Variables["id"] != "acc-1" {
			t.Fatalf("variables = %#v, want id=acc-1", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"updateAccount":{"account":{"id":"acc-1","displayName":"New","displayBalance":100}}}}`), nil
	})

	accountName = ""
	accountBalance = 0
	_ = accountsUpdateCmd.Flags().Set("name", "New")
	_ = accountsUpdateCmd.Flags().Set("balance", "100")
	out := captureStdout(t, func() {
		accountsUpdateCmd.Run(accountsUpdateCmd, []string{"acc-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"display_name":"New"`) {
		t.Fatalf("output missing display name = %q", out)
	}
	if !strings.Contains(out, `"display_balance":100`) {
		t.Fatalf("output missing balance = %q", out)
	}
}

func testAccountsDeleteJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, accountsDeleteCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "DeleteAccount" {
			t.Fatalf("operation = %q, want DeleteAccount", gqlReq.OperationName)
		}
		if gqlReq.Variables["id"] != "acc-1" {
			t.Fatalf("variables = %#v, want id=acc-1", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"deleteAccount":{"ok":true}}}`), nil
	})

	out := captureStdout(t, func() {
		accountsDeleteCmd.Run(accountsDeleteCmd, []string{"acc-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.delete"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"status":"deleted"`) {
		t.Fatalf("output missing status = %q", out)
	}
}

func testAccountsCreateManualJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, accountsCreateManualCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "CreateManualAccount" {
			t.Fatalf("operation = %q, want CreateManualAccount", gqlReq.OperationName)
		}
		if gqlReq.Variables["name"] != "Savings" {
			t.Fatalf("variables = %#v, want name=Savings", gqlReq.Variables)
		}
		if gqlReq.Variables["type"] != "cash" {
			t.Fatalf("variables = %#v, want type=cash", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"createManualAccount":{"account":{"id":"a2","displayName":"Savings","displayBalance":10}}}}`), nil
	})

	accountName = ""
	accountType = ""
	accountBalance = 0
	_ = accountsCreateManualCmd.Flags().Set("name", "Savings")
	_ = accountsCreateManualCmd.Flags().Set("type", "cash")
	_ = accountsCreateManualCmd.Flags().Set("balance", "10")
	out := captureStdout(t, func() {
		accountsCreateManualCmd.Run(accountsCreateManualCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.create-manual"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"display_name":"Savings"`) {
		t.Fatalf("output missing display name = %q", out)
	}
	if !strings.Contains(out, `"display_balance":10`) {
		t.Fatalf("output missing balance = %q", out)
	}
}

func testAccountsUploadHistoryJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")

	// Create a temp CSV file for upload.
	csvPath := filepath.Join(dir, "history.csv")
	if err := os.WriteFile(csvPath, []byte("date,balance\n2026-01-01,100\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	exitCode := withWriteCommandTestDefaults(t, sessionPath, accountsUploadHistoryCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != "POST" {
			t.Fatalf("method = %q, want POST", req.Method)
		}
		if !strings.Contains(req.URL.Path, "account-balance-history") {
			t.Fatalf("path = %q, want account-balance-history", req.URL.Path)
		}
		return testutil.JSONResponse(`{"ok":true}`), nil
	})

	out := captureStdout(t, func() {
		accountsUploadHistoryCmd.Run(accountsUploadHistoryCmd, []string{"acc-1", csvPath})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.upload-history"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"status":"uploaded"`) {
		t.Fatalf("output missing status = %q", out)
	}
}

func testAccountsRecentBalancesJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, accountsRecentBalancesCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetAccountRecentBalances" {
			t.Fatalf("operation = %q, want GetAccountRecentBalances", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"accounts":[{"id":"a1","displayName":"Checking","type":{"group":"asset"},"recentBalances":[1,2,3]}]}}`), nil
	})

	historyFrom = ""
	_ = accountsRecentBalancesCmd.Flags().Set("from", "2026-05-01")
	out := captureStdout(t, func() {
		accountsRecentBalancesCmd.Run(accountsRecentBalancesCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.recent-balances"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"display_name":"Checking"`) {
		t.Fatalf("output missing display name = %q", out)
	}
	if !strings.Contains(out, `"account_type_group":"asset"`) {
		t.Fatalf("output missing account type group = %q", out)
	}
}

func testAccountsSnapshotsJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, accountsSnapshotsCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetSnapshotsByAccountType" {
			t.Fatalf("operation = %q, want GetSnapshotsByAccountType", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"snapshotsByAccountType":[{"accountType":"bank","month":"2026-05","balance":1}],"accountTypes":[{"name":"bank","group":"asset"}]}}`), nil
	})

	historyFrom = ""
	timeframe = ""
	_ = accountsSnapshotsCmd.Flags().Set("from", "2025-06-01")
	_ = accountsSnapshotsCmd.Flags().Set("timeframe", "month")
	out := captureStdout(t, func() {
		accountsSnapshotsCmd.Run(accountsSnapshotsCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.snapshots"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"accountType":"bank"`) {
		t.Fatalf("output missing account type = %q", out)
	}
}

func testAccountsAggregateSnapshotsJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, accountsAggregateSnapshotsCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetAggregateSnapshots" {
			t.Fatalf("operation = %q, want GetAggregateSnapshots", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"aggregateSnapshots":[{"date":"2026-05-01","balance":1}]}}`), nil
	})

	historyFrom = ""
	historyTo = ""
	accountType = ""
	_ = accountsAggregateSnapshotsCmd.Flags().Set("from", "2025-01-01")
	_ = accountsAggregateSnapshotsCmd.Flags().Set("to", "2026-05-31")
	_ = accountsAggregateSnapshotsCmd.Flags().Set("type", "bank")
	out := captureStdout(t, func() {
		accountsAggregateSnapshotsCmd.Run(accountsAggregateSnapshotsCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"accounts.aggregate-snapshots"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"date":"2026-05-01"`) {
		t.Fatalf("output missing date = %q", out)
	}
	if !strings.Contains(out, `"balance":1`) {
		t.Fatalf("output missing balance = %q", out)
	}
}

func testNetworthJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, networthCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetAggregateSnapshots" {
			t.Fatalf("operation = %q, want GetAggregateSnapshots", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"aggregateSnapshots":[{"date":"2026-05-01","balance":1}]}}`), nil
	})

	historyFrom = ""
	historyTo = ""
	accountType = ""
	_ = networthCmd.Flags().Set("from", "2025-01-01")
	_ = networthCmd.Flags().Set("to", "2026-05-31")
	_ = networthCmd.Flags().Set("type", "bank")
	out := captureStdout(t, func() {
		networthCmd.Run(networthCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"networth"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"date":"2026-05-01"`) {
		t.Fatalf("output missing date = %q", out)
	}
}
