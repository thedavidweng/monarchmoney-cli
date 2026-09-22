package cli

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestDebt(t *testing.T) {
	t.Run("paydown", testDebtPaydownJSON)
}

func testDebtPaydownJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, debtPaydownCmd)
	saveTestSession(t, sessionPath)

	_ = debtPaydownCmd.Flags().Set("method", "planned")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetDebtPaydown" {
			t.Fatalf("operation = %q, want GetDebtPaydown", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"debtAccounts":[
			{"id":"a1","displayName":"Card","displayBalance":100,"apr":19.99,"interestRate":null,"minimumPayment":25,"plannedPayment":50,"excludeFromDebtPaydown":false}
		],"debtPaydownPlan":{"currentDebtPrincipal":100,"projectedInterest":5,"projectedTotal":105,"debtFreeDate":"2027-01-01","adjustedDebtFreeDate":"","adjustedProjectedInterest":0,"adjustedProjectedTotal":0,"debtAccountProjections":[]}}}`), nil
	})

	out := captureStdout(t, func() {
		debtPaydownCmd.Run(debtPaydownCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"debt.paydown"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"current_principal":100`) {
		t.Fatalf("output missing principal = %q", out)
	}
}
