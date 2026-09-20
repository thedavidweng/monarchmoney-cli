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

func TestInvestments(t *testing.T) {
	t.Run("portfolio", testInvestmentsPortfolio)
	t.Run("portfolio_api_error", testInvestmentsPortfolioAPIError)
	t.Run("performance", testInvestmentsPerformance)
	t.Run("accounts", testInvestmentsAccounts)
	t.Run("holdings_list", testInvestmentsHoldingsList)
	t.Run("holding_show", testInvestmentsHoldingShow)
	t.Run("securities", testInvestmentsSecurities)
	t.Run("security", testInvestmentsSecurity)
	t.Run("holdings_create", testInvestmentsHoldingsCreate)
	t.Run("holdings_update", testInvestmentsHoldingsUpdate)
	t.Run("holdings_delete", testInvestmentsHoldingsDelete)
}

const testInvestmentHoldingFixture = `{"id":"h-1","quantity":10,"totalValue":1500.0,"security":{"id":"s-1","name":"Apple Inc","ticker":"AAPL","currentPrice":150.0},"holdings":[{"id":"h-1","name":"Apple","ticker":"AAPL","quantity":10,"value":1500.0,"account":{"id":"a-1","displayName":"Brokerage"}}]}`

func testInvestmentsPortfolio(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, investmentsPortfolioCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return testutil.JSONResponse(`{"data":{"portfolio":{"performance":{"totalValue":1000,"totalChangePercent":0.12,"totalChangeDollars":120},"aggregateHoldings":{"edges":[{"node":{"id":"node-1","quantity":2,"basis":400,"totalValue":1000,"security":{"id":"sec-1","ticker":"ABC","name":"ABC Fund","currentPrice":500},"holdings":[]}}]}}}}`), nil
	})

	investmentFrom = ""
	investmentTo = ""
	investmentAccountIDs = nil
	_ = investmentsPortfolioCmd.Flags().Set("from", "2026-01-01")
	_ = investmentsPortfolioCmd.Flags().Set("to", "2026-05-01")
	_ = investmentsPortfolioCmd.Flags().Set("account-id", "acc-1")
	out := captureStdout(t, func() {
		investmentsPortfolioCmd.Run(investmentsPortfolioCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"investments.portfolio"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"total_value":1000`) {
		t.Fatalf("output missing total_value = %q", out)
	}
	if !strings.Contains(out, `"ticker":"ABC"`) {
		t.Fatalf("output missing ticker = %q", out)
	}
}

func testInvestmentsPortfolioAPIError(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, investmentsPortfolioCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	})

	out := captureStdout(t, func() {
		investmentsPortfolioCmd.Run(investmentsPortfolioCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want API failure; output=%q", out)
	}
	if !strings.Contains(out, `"API_ERROR"`) {
		t.Fatalf("output = %q, want API_ERROR", out)
	}
}

func testInvestmentsPerformance(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, investmentsPerformanceCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return testutil.JSONResponse(`{"data":{"securityHistoricalPerformance":[{"security":{"id":"sec-1","ticker":"ABC","name":"ABC Fund"},"historicalChart":[{"date":"2026-01-01","returnPercent":0.1}]}]}}`), nil
	})

	investmentSecurityIDs = nil
	investmentFrom = ""
	investmentTo = ""
	investmentIncludeValues = false
	_ = investmentsPerformanceCmd.Flags().Set("security-id", "sec-1")
	_ = investmentsPerformanceCmd.Flags().Set("from", "2026-01-01")
	_ = investmentsPerformanceCmd.Flags().Set("to", "2026-05-01")
	out := captureStdout(t, func() {
		investmentsPerformanceCmd.Run(investmentsPerformanceCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"investments.performance"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"ticker":"ABC"`) {
		t.Fatalf("output missing ticker = %q", out)
	}
}

func testInvestmentsAccounts(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, investmentsAccountsCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_GetInvestmentsAccounts" {
			t.Fatalf("operation = %q, want Web_GetInvestmentsAccounts", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"accounts":[{"id":"a-1","displayName":"Brokerage","includeInNetWorth":true}]}}`), nil
	})

	out := captureStdout(t, func() {
		investmentsAccountsCmd.Run(investmentsAccountsCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"investments.accounts"`) || !strings.Contains(out, "Brokerage") {
		t.Fatalf("output = %q", out)
	}
}

func testInvestmentsHoldingsList(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, investmentsHoldingsListCmd)
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
		return testutil.JSONResponse(`{"data":{"portfolio":{"aggregateHoldings":{"edges":[{"node":` + testInvestmentHoldingFixture + `}]}}}}`), nil
	})

	out := captureStdout(t, func() {
		investmentsHoldingsListCmd.Run(investmentsHoldingsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"investments.holdings.list"`) || !strings.Contains(out, "AAPL") {
		t.Fatalf("output = %q", out)
	}
}

func testInvestmentsHoldingShow(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, investmentsHoldingShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return testutil.JSONResponse(`{"data":{"portfolio":{"aggregateHoldings":{"edges":[{"node":` + testInvestmentHoldingFixture + `}]}}}}`), nil
	})

	out := captureStdout(t, func() {
		investmentsHoldingShowCmd.Run(investmentsHoldingShowCmd, []string{"h-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"investments.holding.show"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testInvestmentsSecurities(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, investmentsSecuritiesCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_SearchSecurities" {
			t.Fatalf("operation = %q, want Web_SearchSecurities", gqlReq.OperationName)
		}
		if gqlReq.Variables["search"] != "Apple" {
			t.Fatalf("search = %v", gqlReq.Variables["search"])
		}
		return testutil.JSONResponse(`{"data":{"securities":[{"id":"s-1","name":"Apple Inc","ticker":"AAPL","type":"stock","typeDisplay":"Stock","currentPrice":150.0}]}}`), nil
	})

	out := captureStdout(t, func() {
		investmentsSecuritiesCmd.Run(investmentsSecuritiesCmd, []string{"Apple"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"investments.securities"`) || !strings.Contains(out, "AAPL") {
		t.Fatalf("output = %q", out)
	}
}

func testInvestmentsSecurity(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, investmentsSecurityCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetHoldingDetailsFormSecurityDetails" {
			t.Fatalf("operation = %q, want GetHoldingDetailsFormSecurityDetails", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"security":{"id":"s-1","name":"Apple Inc","ticker":"AAPL","type":"stock","typeDisplay":"Stock","currentPrice":150.0}}}`), nil
	})

	out := captureStdout(t, func() {
		investmentsSecurityCmd.Run(investmentsSecurityCmd, []string{"s-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"investments.security"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testInvestmentsHoldingsCreate(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, investmentsHoldingsCreateCmd)
	saveTestSession(t, sessionPath)

	_ = investmentsHoldingsCreateCmd.Flags().Set("account", "a-1")
	_ = investmentsHoldingsCreateCmd.Flags().Set("security", "s-1")
	_ = investmentsHoldingsCreateCmd.Flags().Set("quantity", "10")
	t.Cleanup(func() {
		_ = investmentsHoldingsCreateCmd.Flags().Set("account", "")
		_ = investmentsHoldingsCreateCmd.Flags().Set("security", "")
		_ = investmentsHoldingsCreateCmd.Flags().Set("quantity", "0")
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
		case "Common_CreateManualHolding":
			input, _ := gqlReq.Variables["input"].(map[string]any)
			if input["accountId"] != "a-1" || input["securityId"] != "s-1" || input["quantity"] != float64(10) {
				t.Fatalf("create input = %v", input)
			}
			return testutil.JSONResponse(`{"data":{"createManualHolding":{"holding":{"id":"h-9","ticker":"AAPL"},"errors":null}}}`), nil
		case "Web_GetHoldings":
			return testutil.JSONResponse(`{"data":{"portfolio":{"aggregateHoldings":{"edges":[{"node":{"id":"h-9","quantity":10,"totalValue":1500.0,"security":{"id":"s-1","name":"Apple Inc","ticker":"AAPL","currentPrice":150.0},"holdings":[{"id":"h-9","name":"Apple","ticker":"AAPL","quantity":10,"value":1500.0,"account":{"id":"a-1","displayName":"Brokerage"}}]}}]}}}}`), nil
		default:
			t.Fatalf("operation = %q, want manual holding create ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		investmentsHoldingsCreateCmd.Run(investmentsHoldingsCreateCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"investments.holdings.create"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testInvestmentsHoldingsUpdate(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, investmentsHoldingsUpdateCmd)
	saveTestSession(t, sessionPath)

	_ = investmentsHoldingsUpdateCmd.Flags().Set("quantity", "20")
	t.Cleanup(func() { _ = investmentsHoldingsUpdateCmd.Flags().Set("quantity", "0") })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_UpdateHolding":
			return testutil.JSONResponse(`{"data":{"updateHolding":{"holding":{"id":"h-1"},"errors":null}}}`), nil
		case "Web_GetHoldings":
			return testutil.JSONResponse(`{"data":{"portfolio":{"aggregateHoldings":{"edges":[{"node":` + testInvestmentHoldingFixture + `}]}}}}`), nil
		default:
			t.Fatalf("operation = %q, want manual holding update ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		investmentsHoldingsUpdateCmd.Run(investmentsHoldingsUpdateCmd, []string{"h-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"investments.holdings.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testInvestmentsHoldingsDelete(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, investmentsHoldingsDeleteCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_DeleteHolding" {
			t.Fatalf("operation = %q, want Common_DeleteHolding", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"deleteHolding":{"deleted":true,"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		investmentsHoldingsDeleteCmd.Run(investmentsHoldingsDeleteCmd, []string{"h-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"investments.holdings.delete"`) {
		t.Fatalf("output missing command = %q", out)
	}
}
