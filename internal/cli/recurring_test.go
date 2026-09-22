package cli

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestRecurring(t *testing.T) {
	t.Run("list", testRecurringListJSON)
	t.Run("update", testRecurringUpdateJSON)
	t.Run("streams", testRecurringStreamsJSON)
	t.Run("show", testRecurringShowJSON)
	t.Run("summary", testRecurringSummaryJSON)
	t.Run("create", testRecurringCreateJSON)
	t.Run("stream_update", testRecurringStreamUpdateJSON)
	t.Run("remove", testRecurringRemoveJSON)
	t.Run("review", testRecurringReviewJSON)
	t.Run("review_bad_status", testRecurringReviewBadStatus)
}

const testRecurringStreamFixture = `{"id":"rs-1","reviewStatus":"active","frequency":"monthly","amount":15.99,"baseDate":"2026-01-15","isActive":true,"isApproximate":false,"name":"Netflix","merchant":{"id":"m-1","name":"Netflix"}}`

func testRecurringListJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, recurringListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_GetUpcomingRecurringTransactionItems" {
			t.Fatalf("operation = %q, want Web_GetUpcomingRecurringTransactionItems", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"recurringTransactionItems":[
			{"stream":{"id":"rec-1","frequency":"monthly","amount":15.99,"isApproximate":false,"merchant":{"id":"m-1","name":"Netflix","logoUrl":""}},"date":"2026-06-15","isPast":false,"transactionId":"","amount":15.99,"amountDiff":0,"category":{"id":"cat-1","name":"Entertainment"},"account":{"id":"acc-1","displayName":"Checking"}},
			{"stream":{"id":"rec-2","frequency":"weekly","amount":50,"isApproximate":false,"merchant":{"id":"m-2","name":"Gym","logoUrl":""}},"date":"2026-06-16","isPast":false,"transactionId":"","amount":50,"amountDiff":0,"category":{"id":"cat-2","name":"Health"},"account":{"id":"acc-1","displayName":"Checking"}}
		]}}`), nil
	})

	out := captureStdout(t, func() {
		recurringListCmd.Run(recurringListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"recurring.list"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"merchant":"Netflix"`) {
		t.Fatalf("output missing Netflix = %q", out)
	}
	if !strings.Contains(out, `"frequency":"monthly"`) {
		t.Fatalf("output missing frequency = %q", out)
	}
	if !strings.Contains(out, `"amount":15.99`) {
		t.Fatalf("output missing amount = %q", out)
	}
}

func testRecurringUpdateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, recurringUpdateCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "UpdateRecurringTransaction" {
			t.Fatalf("operation = %q, want UpdateRecurringTransaction", gqlReq.OperationName)
		}
		if gqlReq.Variables["id"] != "rec-1" {
			t.Fatalf("variables id = %v, want rec-1", gqlReq.Variables["id"])
		}
		if gqlReq.Variables["amount"] != float64(19.99) {
			t.Fatalf("variables amount = %v, want 19.99", gqlReq.Variables["amount"])
		}
		return testutil.JSONResponse(`{"data":{"updateRecurringTransaction":{"recurringTransaction":{"id":"rec-1","amount":19.99}}}}`), nil
	})

	recurringAmount = 0
	_ = recurringUpdateCmd.Flags().Set("amount", "19.99")
	out := captureStdout(t, func() {
		recurringUpdateCmd.Run(recurringUpdateCmd, []string{"rec-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"recurring.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, "rec-1") {
		t.Fatalf("output missing ID = %q", out)
	}
}

func testRecurringStreamsJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, recurringStreamsCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_GetAllRecurringTransactionItems" {
			t.Fatalf("operation = %q, want Common_GetAllRecurringTransactionItems", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"recurringTransactionStreams":[{"stream":` + testRecurringStreamFixture + `,"nextForecastedTransaction":{"date":"2026-06-15","amount":15.99},"category":{"id":"cat-1","name":"Entertainment"},"account":{"id":"acc-1","displayName":"Checking"}}]}}`), nil
	})

	out := captureStdout(t, func() {
		recurringStreamsCmd.Run(recurringStreamsCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"recurring.streams"`) || !strings.Contains(out, "Netflix") {
		t.Fatalf("output = %q", out)
	}
}

func testRecurringShowJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, recurringShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return testutil.JSONResponse(`{"data":{"recurringTransactionStreams":[{"stream":` + testRecurringStreamFixture + `,"nextForecastedTransaction":null,"category":null,"account":null}]}}`), nil
	})

	out := captureStdout(t, func() {
		recurringShowCmd.Run(recurringShowCmd, []string{"rs-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"recurring.show"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testRecurringSummaryJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, recurringSummaryCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_GetAggregatedRecurringItems" {
			t.Fatalf("operation = %q, want Common_GetAggregatedRecurringItems", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"aggregatedRecurringItems":{"aggregatedSummary":{"expense":{"completed":100,"remaining":50,"total":150,"count":3},"income":{"completed":2000,"remaining":0,"total":2000}}}}}`), nil
	})

	out := captureStdout(t, func() {
		recurringSummaryCmd.Run(recurringSummaryCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"recurring.summary"`) || !strings.Contains(out, `"expense_total":150`) {
		t.Fatalf("output = %q", out)
	}
}

func testRecurringCreateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, recurringCreateCmd)
	saveTestSession(t, sessionPath)

	_ = recurringCreateCmd.Flags().Set("merchant", "m-1")
	_ = recurringCreateCmd.Flags().Set("frequency", "monthly")
	_ = recurringCreateCmd.Flags().Set("amount", "15.99")
	t.Cleanup(func() {
		_ = recurringCreateCmd.Flags().Set("merchant", "")
		_ = recurringCreateCmd.Flags().Set("frequency", "")
		_ = recurringCreateCmd.Flags().Set("amount", "0")
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
		case "Common_RecurringUpdateMerchant":
			input, _ := gqlReq.Variables["input"].(map[string]any)
			if input["merchantId"] != "m-1" {
				t.Fatalf("recurrence input = %v", input)
			}
			return testutil.JSONResponse(`{"data":{"updateMerchant":{"merchant":{"id":"m-1"},"errors":null}}}`), nil
		case "Common_GetAllRecurringTransactionItems":
			return testutil.JSONResponse(`{"data":{"recurringTransactionStreams":[{"stream":` + testRecurringStreamFixture + `,"nextForecastedTransaction":null,"category":null,"account":null}]}}`), nil
		default:
			t.Fatalf("operation = %q, want recurring create ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		recurringCreateCmd.Run(recurringCreateCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"recurring.create"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testRecurringStreamUpdateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, recurringStreamUpdateCmd)
	saveTestSession(t, sessionPath)

	_ = recurringStreamUpdateCmd.Flags().Set("amount", "19.99")
	t.Cleanup(func() { _ = recurringStreamUpdateCmd.Flags().Set("amount", "0") })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_GetAllRecurringTransactionItems":
			return testutil.JSONResponse(`{"data":{"recurringTransactionStreams":[{"stream":` + testRecurringStreamFixture + `,"nextForecastedTransaction":null,"category":null,"account":null}]}}`), nil
		case "Common_RecurringUpdateMerchant":
			return testutil.JSONResponse(`{"data":{"updateMerchant":{"merchant":{"id":"m-1"},"errors":null}}}`), nil
		default:
			t.Fatalf("operation = %q, want recurring stream-update ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		recurringStreamUpdateCmd.Run(recurringStreamUpdateCmd, []string{"rs-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"recurring.stream-update"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testRecurringRemoveJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, recurringRemoveCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_MarkAsNotRecurring" {
			t.Fatalf("operation = %q, want Common_MarkAsNotRecurring", gqlReq.OperationName)
		}
		if gqlReq.Variables["streamId"] != "rs-1" {
			t.Fatalf("remove variables = %v", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"markStreamAsNotRecurring":{"success":true,"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		recurringRemoveCmd.Run(recurringRemoveCmd, []string{"rs-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"recurring.remove"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func TestRecurringHumanOutput(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath,
		recurringStreamsCmd, recurringShowCmd, recurringSummaryCmd, recurringCreateCmd,
		recurringStreamUpdateCmd, recurringRemoveCmd)
	saveTestSession(t, sessionPath)

	oldJSON := jsonMode
	jsonMode = false
	t.Cleanup(func() { jsonMode = oldJSON })

	_ = recurringCreateCmd.Flags().Set("merchant", "m-1")
	_ = recurringCreateCmd.Flags().Set("frequency", "monthly")
	_ = recurringCreateCmd.Flags().Set("amount", "15.99")
	_ = recurringStreamUpdateCmd.Flags().Set("amount", "19.99")
	t.Cleanup(func() {
		_ = recurringCreateCmd.Flags().Set("merchant", "")
		_ = recurringCreateCmd.Flags().Set("frequency", "")
		_ = recurringCreateCmd.Flags().Set("amount", "0")
		_ = recurringStreamUpdateCmd.Flags().Set("amount", "0")
	})

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_GetAllRecurringTransactionItems":
			return testutil.JSONResponse(`{"data":{"recurringTransactionStreams":[{"stream":` + testRecurringStreamFixture + `,"nextForecastedTransaction":{"date":"2026-06-15","amount":15.99},"category":null,"account":null}]}}`), nil
		case "Common_GetAggregatedRecurringItems":
			return testutil.JSONResponse(`{"data":{"aggregatedRecurringItems":{"aggregatedSummary":{"expense":{"completed":100,"remaining":50,"total":150,"count":3},"income":{"completed":0,"remaining":0,"total":0}}}}}`), nil
		case "Common_RecurringUpdateMerchant":
			return testutil.JSONResponse(`{"data":{"updateMerchant":{"merchant":{"id":"m-1"},"errors":null}}}`), nil
		case "Common_MarkAsNotRecurring":
			return testutil.JSONResponse(`{"data":{"markStreamAsNotRecurring":{"success":true,"errors":null}}}`), nil
		default:
			t.Fatalf("operation = %q", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		recurringStreamsCmd.Run(recurringStreamsCmd, nil)
		recurringShowCmd.Run(recurringShowCmd, []string{"rs-1"})
		recurringSummaryCmd.Run(recurringSummaryCmd, nil)
		recurringCreateCmd.Run(recurringCreateCmd, nil)
		recurringStreamUpdateCmd.Run(recurringStreamUpdateCmd, []string{"rs-1"})
		recurringRemoveCmd.Run(recurringRemoveCmd, []string{"rs-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	for _, want := range []string{"Netflix", "Expense:", "created recurring stream", "updated recurring stream", "removed recurring stream"} {
		if !strings.Contains(out, want) {
			t.Fatalf("human output missing %q in %q", want, out)
		}
	}
}

func testRecurringReviewJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, recurringReviewCmd)
	saveTestSession(t, sessionPath)

	_ = recurringReviewCmd.Flags().Set("status", "approved")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_ReviewStream" {
			t.Fatalf("operation = %q, want Web_ReviewStream", gqlReq.OperationName)
		}
		input, _ := gqlReq.Variables["input"].(map[string]any)
		if input["streamId"] != "rs-1" || input["reviewStatus"] != "approved" {
			t.Fatalf("input = %#v, want stream rs-1 approved", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"reviewRecurringStream":{"stream":{"id":"rs-1","reviewStatus":"approved"},"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		recurringReviewCmd.Run(recurringReviewCmd, []string{"rs-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"recurring.review"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testRecurringReviewBadStatus(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, recurringReviewCmd)
	saveTestSession(t, sessionPath)

	_ = recurringReviewCmd.Flags().Set("status", "bogus")

	out := captureStdout(t, func() {
		recurringReviewCmd.Run(recurringReviewCmd, []string{"rs-1"})
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "--status must be approved, ignored, or pending") {
		t.Fatalf("output = %q, want status guidance", out)
	}
	_ = recurringReviewCmd.Flags().Set("status", "approved")
}
