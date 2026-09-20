package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestReceipts(t *testing.T) {
	t.Run("upload", testReceiptsUpload)
	t.Run("list", testReceiptsList)
	t.Run("list_invalid_source", testReceiptsListInvalidSource)
	t.Run("show", testReceiptsShow)
	t.Run("download", testReceiptsDownload)
	t.Run("delete", testReceiptsDelete)
	t.Run("match", testReceiptsMatch)
	t.Run("unmatch", testReceiptsUnmatch)
	t.Run("update", testReceiptsUpdate)
	t.Run("settings", testReceiptsSettings)
	t.Run("settings_update", testReceiptsSettingsUpdate)
}

const testReceiptFixture = `{"id":"sync-1","vendor":"user_import","status":"pending_matches","startedAt":"2026-05-01T00:00:00Z","endedAt":null,"createdAt":"2026-05-01T00:00:00Z","updatedAt":"2026-05-01T00:00:00Z","orders":[{"id":"order-1","merchantName":"Whole Foods","vendor":"user_import","vendorOrderId":"o-1","date":"2026-05-01","totalForProducts":42.5,"shipping":0,"deliveryFee":0,"additionalCharges":0,"adjustmentsAmount":0,"totalBeforeTax":42.5,"tax":3.5,"tip":0,"giftCardAmount":0,"grandTotal":46.0,"displayStatus":"pending","retailLineItems":[{"id":"li-1","title":"Milk","quantity":1,"price":5.0,"total":5.0,"isAssociatedToRetailTransaction":false,"category":{"id":"cat-1","name":"Groceries","icon":"cart"}}],"retailTransactions":[{"id":"rt-1","date":"2026-05-01","total":46.0,"transactionType":"payment","transactionUpdateSkipped":false,"transaction":{"id":"tx-9","isManual":false,"hasSplitTransactions":false,"merchant":{"id":"m-1","name":"Whole Foods","logoUrl":"https://example.com/logo.png"}}}]}],"attachments":[{"id":"att-1","storageId":"s-1","filename":"receipt.jpg","extension":"jpg","sizeBytes":1024,"originalAssetUrl":"https://example.com/receipt.jpg","thumbnailUrl":"https://example.com/receipt-thumb.jpg"}]}`

func testReceiptsUpload(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, receiptsUploadCmd)
	saveTestSession(t, sessionPath)

	receiptPath := filepath.Join(dir, "receipt.jpg")
	if err := os.WriteFile(receiptPath, []byte("jpg"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/retail-sync/sync-1/files" {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if !strings.Contains(string(body), `filename="receipt.jpg"`) || !strings.Contains(string(body), `"vendor":"user_import"`) {
				t.Fatalf("retail sync body missing fields: %s", body)
			}
			return testutil.JSONResponse(`{}`), nil
		}
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_CreateBulkRetailSync":
			return testutil.JSONResponse(`{"data":{"createBulkRetailSync":{"retailSyncs":[{"id":"sync-1","vendor":"user_import","status":"created"}],"errors":null}}}`), nil
		case "Common_StartRetailSync":
			return testutil.JSONResponse(`{"data":{"startRetailSync":{"retailSync":{"id":"sync-1","vendor":"user_import","status":"started","startedAt":"2026-05-01T00:00:00Z"},"errors":null}}}`), nil
		default:
			t.Fatalf("operation = %q, want retail sync ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		receiptsUploadCmd.Run(receiptsUploadCmd, []string{receiptPath})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"receipts.upload"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"status":"started"`) {
		t.Fatalf("output missing status = %q", out)
	}
}

func testReceiptsList(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, receiptsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_RetailSyncsQueryWithTotal" {
			t.Fatalf("operation = %q, want Common_RetailSyncsQueryWithTotal", gqlReq.OperationName)
		}
		filters, _ := gqlReq.Variables["filters"].(map[string]any)
		vendor, _ := filters["vendor"].(string)
		if vendor != "user_import" && vendor != "email_import" {
			t.Fatalf("filters.vendor = %q, want user_import or email_import", vendor)
		}
		if vendor == "email_import" {
			return testutil.JSONResponse(`{"data":{"retailSyncsWithTotal":{"totalCount":0,"results":[]}}}`), nil
		}
		return testutil.JSONResponse(`{"data":{"retailSyncsWithTotal":{"totalCount":1,"results":[` + testReceiptFixture + `]}}}`), nil
	})

	out := captureStdout(t, func() {
		receiptsListCmd.Run(receiptsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"receipts.list"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, "sync-1") || !strings.Contains(out, "Whole Foods") {
		t.Fatalf("output missing receipt = %q", out)
	}
}

func testReceiptsListInvalidSource(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, receiptsListCmd)
	saveTestSession(t, sessionPath)

	old := receiptSource
	receiptSource = "carrier-pigeon"
	t.Cleanup(func() { receiptSource = old })

	out := captureStdout(t, func() {
		receiptsListCmd.Run(receiptsListCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want non-zero; output=%q", out)
	}
	if !strings.Contains(out, "--source must be upload or email") {
		t.Fatalf("output missing validation error = %q", out)
	}
}

func testReceiptsShow(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, receiptsShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_RetailSyncQuery" {
			t.Fatalf("operation = %q, want Common_RetailSyncQuery", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"retailSync":{"id":"sync-1","vendor":"user_import","status":"completed","startedAt":"2026-05-01T00:00:00Z","endedAt":null,"createdAt":"2026-05-01T00:00:00Z","updatedAt":"2026-05-01T00:00:00Z","orders":[],"attachments":[]}}}`), nil
	})

	out := captureStdout(t, func() {
		receiptsShowCmd.Run(receiptsShowCmd, []string{"sync-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"receipts.show"`) || !strings.Contains(out, "sync-1") {
		t.Fatalf("output = %q", out)
	}
}

func testReceiptsDownload(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, receiptsDownloadCmd)
	saveTestSession(t, sessionPath)

	outPath := filepath.Join(dir, "out.jpg")
	old := outputFile
	outputFile = outPath
	t.Cleanup(func() { outputFile = old })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "example.com" {
			return testutil.JSONResponse(`fake-image-bytes`), nil
		}
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_RetailSyncQuery" {
			t.Fatalf("operation = %q, want Common_RetailSyncQuery", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"retailSync":` + testReceiptFixture + `}}`), nil
	})

	out := captureStdout(t, func() {
		receiptsDownloadCmd.Run(receiptsDownloadCmd, []string{"sync-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"receipts.download"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("output file missing: %v", err)
	}
}

func testReceiptsDelete(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, receiptsDeleteCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_DeleteRetailSync" {
			t.Fatalf("operation = %q, want Common_DeleteRetailSync", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"deleteUnmatchedRetailSync":{"success":true,"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		receiptsDeleteCmd.Run(receiptsDeleteCmd, []string{"sync-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"receipts.delete"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testReceiptsMatch(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, receiptsMatchCmd)
	saveTestSession(t, sessionPath)

	old := receiptTransactionID
	receiptTransactionID = "tx-9"
	t.Cleanup(func() { receiptTransactionID = old })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_RetailSyncQuery":
			return testutil.JSONResponse(`{"data":{"retailSync":` + testReceiptFixture + `}}`), nil
		case "Common_MatchRetailTransaction":
			if gqlReq.Variables["retailTransactionId"] != "rt-1" || gqlReq.Variables["transactionId"] != "tx-9" {
				t.Fatalf("match variables = %v", gqlReq.Variables)
			}
			return testutil.JSONResponse(`{"data":{"matchRetailTransaction":{"retailSync":{"id":"sync-1"},"errors":null}}}`), nil
		default:
			t.Fatalf("operation = %q, want receipt match ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		receiptsMatchCmd.Run(receiptsMatchCmd, []string{"sync-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"receipts.match"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testReceiptsUnmatch(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, receiptsUnmatchCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_RetailSyncQuery":
			return testutil.JSONResponse(`{"data":{"retailSync":` + testReceiptFixture + `}}`), nil
		case "Web_UnmatchRetailTransaction":
			if gqlReq.Variables["retailTransactionId"] != "rt-1" {
				t.Fatalf("unmatch variables = %v", gqlReq.Variables)
			}
			return testutil.JSONResponse(`{"data":{"unmatchRetailTransaction":{"retailSync":{"id":"sync-1","status":"pending_matches"},"errors":null}}}`), nil
		default:
			t.Fatalf("operation = %q, want receipt unmatch ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		receiptsUnmatchCmd.Run(receiptsUnmatchCmd, []string{"sync-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"receipts.unmatch"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testReceiptsUpdate(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, receiptsUpdateCmd)
	saveTestSession(t, sessionPath)

	oldMerchant := receiptMerchant
	receiptMerchant = "Whole Foods Market"
	t.Cleanup(func() { receiptMerchant = oldMerchant })
	if err := receiptsUpdateCmd.Flags().Set("merchant", "Whole Foods Market"); err != nil {
		t.Fatalf("Set merchant flag: %v", err)
	}
	t.Cleanup(func() {
		if err := receiptsUpdateCmd.Flags().Set("merchant", ""); err != nil {
			t.Fatalf("reset merchant flag: %v", err)
		}
	})

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_RetailSyncQuery":
			return testutil.JSONResponse(`{"data":{"retailSync":` + testReceiptFixture + `}}`), nil
		case "Common_UpdateRetailOrder":
			input, _ := gqlReq.Variables["input"].(map[string]any)
			if input["retailOrderId"] != "order-1" || input["merchantName"] != "Whole Foods Market" {
				t.Fatalf("update input = %v", input)
			}
			return testutil.JSONResponse(`{"data":{"updateRetailOrder":{"retailSync":` + testReceiptFixture + `,"errors":null}}}`), nil
		default:
			t.Fatalf("operation = %q, want receipt update ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		receiptsUpdateCmd.Run(receiptsUpdateCmd, []string{"sync-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"receipts.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testReceiptsSettings(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, receiptsSettingsCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_GetRetailExtensionSettings" {
			t.Fatalf("operation = %q, want Common_GetRetailExtensionSettings", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"retailExtensionSettings":{"id":"ext-1","retailVendorSettings":[{"id":"vs-1","vendor":"user_import","shouldCategorizeAndSplitTransactions":true,"shouldUpdateTransactionsNotes":true}]}}}`), nil
	})

	out := captureStdout(t, func() {
		receiptsSettingsCmd.Run(receiptsSettingsCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"receipts.settings"`) || !strings.Contains(out, "user_import") {
		t.Fatalf("output = %q", out)
	}
}

func testReceiptsSettingsUpdate(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, receiptsSettingsUpdateCmd)
	saveTestSession(t, sessionPath)

	if err := receiptsSettingsUpdateCmd.Flags().Set("auto-categorize", "true"); err != nil {
		t.Fatalf("Set flag: %v", err)
	}
	t.Cleanup(func() {
		if err := receiptsSettingsUpdateCmd.Flags().Set("auto-categorize", "false"); err != nil {
			t.Fatalf("reset flag: %v", err)
		}
	})

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_UpdateRetailVendorSettings" {
			t.Fatalf("operation = %q, want Common_UpdateRetailVendorSettings", gqlReq.OperationName)
		}
		input, _ := gqlReq.Variables["input"].(map[string]any)
		if input["vendor"] != "user_import" || input["shouldCategorizeAndSplitTransactions"] != true {
			t.Fatalf("settings input = %v", input)
		}
		return testutil.JSONResponse(`{"data":{"updateRetailVendorSettings":{"retailVendorSettings":{"id":"vs-1","vendor":"user_import","shouldCategorizeAndSplitTransactions":true,"shouldUpdateTransactionsNotes":true},"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		receiptsSettingsUpdateCmd.Run(receiptsSettingsUpdateCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"receipts.settings.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
}
