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

func TestTransactions(t *testing.T) {
	t.Run("list", testTransactionsList)
	t.Run("list_api_error", testTransactionsListAPIError)
	t.Run("show_missing_args", testTransactionsShowMissingArgs)
	t.Run("show", testTransactionsShow)
	t.Run("create", testTransactionsCreate)
	t.Run("update", testTransactionsUpdate)
	t.Run("delete", testTransactionsDelete)
	t.Run("summary", testTransactionsSummary)
	t.Run("export_json", testTransactionsExportJSON)
	t.Run("tags_set", testTransactionsTagsSet)
	t.Run("tags_clear", testTransactionsTagsClear)
	t.Run("tags_add", testTransactionsTagsAdd)
	t.Run("splits", testTransactionsSplits)
	t.Run("bulk_categorize", testTransactionsBulkCategorize)
	t.Run("attachments_list", testTransactionsAttachmentsList)
	t.Run("attachments_upload", testTransactionsAttachmentsUpload)
	t.Run("search", testTransactionsSearch)
}

func testTransactionsList(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, transactionsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetTransactionsList" {
			t.Fatalf("operation = %q, want GetTransactionsList", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"allTransactions":{"results":[{"id":"tx-1","date":"2026-05-08","amount":-20,"merchant":{"name":"Store"},"category":{"name":"Food"},"notes":"lunch","tags":[],"goal":{"id":"","name":""},"account":{"id":"acc-1","displayName":"Checking","order":0,"type":{"group":"depository"}},"ownedByUser":{"displayName":"Test User"}}],"totalCount":1}}}`), nil
	})

	out := captureStdout(t, func() {
		transactionsListCmd.Run(transactionsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.list"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"total":1`) {
		t.Fatalf("output missing total = %q", out)
	}
	if !strings.Contains(out, "Store") {
		t.Fatalf("output missing merchant = %q", out)
	}
}

func testTransactionsListAPIError(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, transactionsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	})

	out := captureStdout(t, func() {
		transactionsListCmd.Run(transactionsListCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want API failure; output=%q", out)
	}
	if !strings.Contains(out, `"API_ERROR"`) {
		t.Fatalf("output = %q, want API_ERROR", out)
	}
}

func testTransactionsShowMissingArgs(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, transactionsShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	})

	out := captureStdout(t, func() {
		transactionsShowCmd.Run(transactionsShowCmd, []string{"tx-1"})
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want API failure; output=%q", out)
	}
	if !strings.Contains(out, `"API_ERROR"`) {
		t.Fatalf("output = %q, want API_ERROR", out)
	}
}

func testTransactionsShow(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, transactionsShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetTransaction" {
			t.Fatalf("operation = %q, want GetTransaction", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"getTransaction":{"id":"tx-1","date":"2026-05-08","amount":-20,"merchant":{"name":"Store"},"category":{"name":"Food"},"notes":"lunch","account":{"id":"acc-1","displayName":"Checking"},"tags":[]}}}`), nil
	})

	out := captureStdout(t, func() {
		transactionsShowCmd.Run(transactionsShowCmd, []string{"tx-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.show"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"id":"tx-1"`) {
		t.Fatalf("output missing id = %q", out)
	}
	if !strings.Contains(out, `"merchant":"Store"`) {
		t.Fatalf("output missing merchant = %q", out)
	}
	if !strings.Contains(out, `"category":"Food"`) {
		t.Fatalf("output missing category = %q", out)
	}
}

func testTransactionsCreate(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, transactionsCreateCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_CreateTransactionMutation" {
			t.Fatalf("operation = %q, want Common_CreateTransactionMutation", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"createTransaction":{"transaction":{"id":"tx-1","amount":-20,"date":"2026-05-08","merchant":{"name":"Store"}}}}}`), nil
	})

	_ = transactionsCreateCmd.Flags().Set("amount", "-20")
	_ = transactionsCreateCmd.Flags().Set("merchant", "Store")
	_ = transactionsCreateCmd.Flags().Set("date", "2026-05-08")
	_ = transactionsCreateCmd.Flags().Set("category", "cat-food")
	_ = transactionsCreateCmd.Flags().Set("account", "acc-1")
	out := captureStdout(t, func() {
		transactionsCreateCmd.Run(transactionsCreateCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.create"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, "tx-1") {
		t.Fatalf("output missing transaction ID = %q", out)
	}
}

func testTransactionsUpdate(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, transactionsUpdateCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_TransactionDrawerUpdateTransaction" {
			t.Fatalf("operation = %q, want Web_TransactionDrawerUpdateTransaction", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"updateTransaction":{"transaction":{"id":"tx-1","amount":0,"date":"","notes":"updated","hideFromReports":false,"needsReview":false,"category":{"name":"Food"},"merchant":{"name":""}}}}}`), nil
	})

	_ = transactionsUpdateCmd.Flags().Set("notes", "updated")
	out := captureStdout(t, func() {
		transactionsUpdateCmd.Run(transactionsUpdateCmd, []string{"tx-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"notes":"updated"`) {
		t.Fatalf("output missing notes = %q", out)
	}
}

func testTransactionsDelete(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, transactionsDeleteCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_DeleteTransactionMutation" {
			t.Fatalf("operation = %q, want Common_DeleteTransactionMutation", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"deleteTransaction":{"deleted":true}}}`), nil
	})

	out := captureStdout(t, func() {
		transactionsDeleteCmd.Run(transactionsDeleteCmd, []string{"tx-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.delete"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"status":"deleted"`) {
		t.Fatalf("output missing status = %q", out)
	}
}

func testTransactionsSummary(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, transactionsSummaryCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetTransactionsPage" {
			t.Fatalf("operation = %q, want GetTransactionsPage", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"aggregates":[{"summary":{"avg":20,"count":1,"max":20,"sum":20,"sumIncome":0,"sumExpense":20}}]}}`), nil
	})

	out := captureStdout(t, func() {
		transactionsSummaryCmd.Run(transactionsSummaryCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.summary"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"sum_expense":20`) {
		t.Fatalf("output missing sum_expense = %q", out)
	}
}

func testTransactionsExportJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, transactionsExportCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		return testutil.JSONResponse(`{"data":{"allTransactions":{"results":[{"id":"tx-1","date":"2026-05-08","amount":-20,"merchant":{"name":"Store"},"category":{"name":"Food"},"notes":"lunch","tags":[],"goal":{"id":"","name":""},"account":{"id":"acc-1","displayName":"Checking","order":0,"type":{"group":"depository"}},"ownedByUser":{"displayName":"Test User"}}],"totalCount":1}}}`), nil
	})

	_ = transactionsExportCmd.Flags().Set("format", "json")
	out := captureStdout(t, func() {
		transactionsExportCmd.Run(transactionsExportCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.export"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, "Store") {
		t.Fatalf("output missing merchant = %q", out)
	}
}

func testTransactionsTagsSet(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, transactionsTagsSetCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_SetTransactionTags" {
			t.Fatalf("operation = %q, want Web_SetTransactionTags", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"setTransactionTags":{}}}`), nil
	})

	tagIDs = nil
	_ = transactionsTagsSetCmd.Flags().Set("tag", "tag-1,tag-2")
	out := captureStdout(t, func() {
		transactionsTagsSetCmd.Run(transactionsTagsSetCmd, []string{"tx-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.tags.set"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"status":"tags set"`) {
		t.Fatalf("output missing status = %q", out)
	}
}

func testTransactionsTagsClear(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, transactionsTagsClearCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_SetTransactionTags" {
			t.Fatalf("operation = %q, want Web_SetTransactionTags", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"setTransactionTags":{}}}`), nil
	})

	out := captureStdout(t, func() {
		transactionsTagsClearCmd.Run(transactionsTagsClearCmd, []string{"tx-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.tags.clear"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"status":"tags cleared"`) {
		t.Fatalf("output missing status = %q", out)
	}
}

func testTransactionsTagsAdd(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, transactionsTagsAddCmd)
	saveTestSession(t, sessionPath)

	callCount := 0
	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		callCount++
		if callCount == 1 {
			if gqlReq.OperationName != "GetTransaction" {
				t.Fatalf("call %d: operation = %q, want GetTransaction", callCount, gqlReq.OperationName)
			}
			return testutil.JSONResponse(`{"data":{"getTransaction":{"id":"tx-1","date":"2026-05-08","amount":-20,"merchant":{"name":"Store"},"category":{"name":"Food"},"notes":"lunch","account":{"id":"acc-1","displayName":"Checking"},"tags":[{"id":"tag-old","name":"existing","color":"#ff0000"}]}}}`), nil
		}
		if gqlReq.OperationName != "Web_SetTransactionTags" {
			t.Fatalf("call %d: operation = %q, want Web_SetTransactionTags", callCount, gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"setTransactionTags":{}}}`), nil
	})

	tagIDs = nil
	_ = transactionsTagsAddCmd.Flags().Set("tag", "tag-new")
	out := captureStdout(t, func() {
		transactionsTagsAddCmd.Run(transactionsTagsAddCmd, []string{"tx-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.tags.add"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"status":"tags added"`) {
		t.Fatalf("output missing status = %q", out)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 API calls, got %d", callCount)
	}
}

func testTransactionsSplits(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, transactionsSplitsCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "TransactionSplitQuery" {
			t.Fatalf("operation = %q, want TransactionSplitQuery", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"getTransaction":{"id":"tx-1","amount":-60,"splitTransactions":[{"id":"split-1","amount":-20,"notes":"part1","merchant":{"name":"Store"},"category":{"name":"Food"}},{"id":"split-2","amount":-40,"notes":"part2","merchant":{"name":"Store"},"category":{"name":"Drinks"}}]}}}`), nil
	})

	out := captureStdout(t, func() {
		transactionsSplitsCmd.Run(transactionsSplitsCmd, []string{"tx-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.splits"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"category":"Food"`) {
		t.Fatalf("output missing category = %q", out)
	}
}

func testTransactionsBulkCategorize(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, transactionsBulkCategorizeCmd)
	saveTestSession(t, sessionPath)

	callCount := 0
	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		callCount++
		if gqlReq.OperationName != "Web_TransactionDrawerUpdateTransaction" {
			t.Fatalf("call %d: operation = %q, want Web_TransactionDrawerUpdateTransaction", callCount, gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"updateTransaction":{"transaction":{"id":"tx-1","amount":0,"date":"","notes":"","hideFromReports":false,"needsReview":false,"category":{"name":"Food"},"merchant":{"name":""}}}}}`), nil
	})

	bulkTxIDs = nil
	bulkCategoryID = ""
	_ = transactionsBulkCategorizeCmd.Flags().Set("id", "tx-1,tx-2")
	_ = transactionsBulkCategorizeCmd.Flags().Set("category-id", "cat-food")
	out := captureStdout(t, func() {
		transactionsBulkCategorizeCmd.Run(transactionsBulkCategorizeCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.bulk-categorize"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"successful":2`) {
		t.Fatalf("output missing successful count = %q", out)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 API calls, got %d", callCount)
	}
}

func testTransactionsAttachmentsList(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, transactionsAttachmentsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetTransaction" {
			t.Fatalf("operation = %q, want GetTransaction", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"getTransaction":{"attachments":[{"id":"att-1","extension":"pdf","filename":"receipt.pdf","originalAssetUrl":"https://example.com/receipt.pdf","sizeBytes":1024}]}}}`), nil
	})

	out := captureStdout(t, func() {
		transactionsAttachmentsListCmd.Run(transactionsAttachmentsListCmd, []string{"tx-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.attachments.list"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, "receipt.pdf") {
		t.Fatalf("output missing filename = %q", out)
	}
}

func testTransactionsAttachmentsUpload(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, transactionsAttachmentsUploadCmd)
	saveTestSession(t, sessionPath)

	tmpFile := filepath.Join(dir, "receipt.pdf")
	_ = os.WriteFile(tmpFile, []byte("pdf"), 0600)

	out := captureStdout(t, func() {
		transactionsAttachmentsUploadCmd.Run(transactionsAttachmentsUploadCmd, []string{"tx-1", tmpFile})
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want FEATURE_UNAVAILABLE; output=%q", out)
	}
	if !strings.Contains(out, `"FEATURE_UNAVAILABLE"`) {
		t.Fatalf("output = %q, want FEATURE_UNAVAILABLE", out)
	}
}

func testTransactionsSearch(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, transactionsSearchCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetTransactionsList" {
			t.Fatalf("operation = %q, want GetTransactionsList", gqlReq.OperationName)
		}
		filters := gqlReq.Variables["filters"].(map[string]any)
		if filters["search"] != "Amazon" {
			t.Fatalf("search = %v, want Amazon", filters["search"])
		}
		return testutil.JSONResponse(`{"data":{"allTransactions":{"results":[{"id":"tx-1","date":"2026-05-08","amount":-50,"merchant":{"name":"Amazon"},"category":{"name":"Shopping"},"notes":"order","tags":[],"goal":{"id":"","name":""},"account":{"id":"acc-1","displayName":"Checking","order":0,"type":{"group":"depository"}},"ownedByUser":{"displayName":"Test User"}}],"totalCount":1}}}`), nil
	})

	out := captureStdout(t, func() {
		transactionsSearchCmd.Run(transactionsSearchCmd, []string{"Amazon"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"transactions.search"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"total":1`) {
		t.Fatalf("output missing total = %q", out)
	}
}
