package cli

import (
	"bytes"
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
}

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
