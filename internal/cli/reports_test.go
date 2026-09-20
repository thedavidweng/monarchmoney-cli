package cli

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestReports(t *testing.T) {
	t.Run("data", testReportsData)
	t.Run("list", testReportsList)
	t.Run("show", testReportsShow)
	t.Run("create", testReportsCreate)
	t.Run("update", testReportsUpdate)
	t.Run("delete", testReportsDelete)
}

func testReportsData(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, reportsDataCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_GetReportsData" {
			t.Fatalf("operation = %q, want Common_GetReportsData", gqlReq.OperationName)
		}
		if gqlReq.Variables["includeCategory"] != true {
			t.Fatalf("variables = %v, want includeCategory=true", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"reports":[{"groupBy":{"date":"2026-01","category":{"id":"c-1","name":"Groceries"}},"summary":{"sum":-420.5,"count":12}}],"aggregates":[{"summary":{"sum":-420.5,"count":12}}]}}`), nil
	})

	out := captureStdout(t, func() {
		reportsDataCmd.Run(reportsDataCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"reports.data"`) || !strings.Contains(out, "Groceries") {
		t.Fatalf("output = %q", out)
	}
}

func testReportsList(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, reportsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_GetReportConfigurations" {
			t.Fatalf("operation = %q, want Web_GetReportConfigurations", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"reportConfigurations":[{"id":"r-1","displayName":"Monthly","reportView":{"timeframe":"month","chartType":"bar","dimensions":["category"]}}]}}`), nil
	})

	out := captureStdout(t, func() {
		reportsListCmd.Run(reportsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"reports.list"`) || !strings.Contains(out, "Monthly") {
		t.Fatalf("output = %q", out)
	}
}

func testReportsShow(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, reportsShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return testutil.JSONResponse(`{"data":{"reportConfigurations":[{"id":"r-1","displayName":"Monthly","reportView":null}]}}`), nil
	})

	out := captureStdout(t, func() {
		reportsShowCmd.Run(reportsShowCmd, []string{"r-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"reports.show"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testReportsCreate(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, reportsCreateCmd)
	saveTestSession(t, sessionPath)

	_ = reportsCreateCmd.Flags().Set("name", "Quarterly")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_CreateReportConfiguration" {
			t.Fatalf("operation = %q, want Web_CreateReportConfiguration", gqlReq.OperationName)
		}
		input, _ := gqlReq.Variables["input"].(map[string]any)
		if input["displayName"] != "Quarterly" {
			t.Fatalf("create input = %v", input)
		}
		return testutil.JSONResponse(`{"data":{"createReportConfiguration":{"reportConfiguration":{"id":"r-2","displayName":"Quarterly"},"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		reportsCreateCmd.Run(reportsCreateCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"reports.create"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testReportsUpdate(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, reportsUpdateCmd)
	saveTestSession(t, sessionPath)

	_ = reportsUpdateCmd.Flags().Set("name", "Monthly v2")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_UpdateReportConfiguration" {
			t.Fatalf("operation = %q, want Web_UpdateReportConfiguration", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"updateReportConfiguration":{"reportConfiguration":{"id":"r-1","displayName":"Monthly v2"},"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		reportsUpdateCmd.Run(reportsUpdateCmd, []string{"r-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"reports.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testReportsDelete(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, reportsDeleteCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_DeleteReportConfiguration" {
			t.Fatalf("operation = %q, want Web_DeleteReportConfiguration", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"deleteReportConfiguration":{"deleted":true,"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		reportsDeleteCmd.Run(reportsDeleteCmd, []string{"r-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"reports.delete"`) {
		t.Fatalf("output missing command = %q", out)
	}
}
