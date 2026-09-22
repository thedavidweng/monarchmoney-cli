package cli

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestWhoami(t *testing.T) {
	t.Run("whoami", testWhoamiJSON)
}

func testWhoamiJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, whoamiCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "GetWhoAmI":
			return testutil.JSONResponse(`{"data":{"me":{"id":"u1","name":"Pat","email":"pat@example.com","timezone":"America/Vancouver","hasPassword":true,"externalAuthProviderNames":[]},"subscription":{"id":"s1","entitlements":["premium"],"hasPremiumEntitlement":true,"isOnFreeTrial":false,"billingPeriod":"yearly","currentPeriodEndsAt":"","trialEndsAt":"","willCancelAtPeriodEnd":false,"paymentSource":"","nextPaymentAmount":0}}}`), nil
		case "ProbeBusinessEntities":
			return testutil.JSONResponse(`{"data":{"businessEntities":[]}}`), nil
		default:
			t.Fatalf("operation = %q, want GetWhoAmI or ProbeBusinessEntities", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		whoamiCmd.Run(whoamiCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"whoami"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"email":"pat@example.com"`) {
		t.Fatalf("output missing email = %q", out)
	}
}
