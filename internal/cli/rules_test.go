package cli

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestRules(t *testing.T) {
	t.Run("list", testRulesListJSON)
	t.Run("update", testRulesUpdateJSON)
	t.Run("delete", testRulesDeleteJSON)
}

func testRulesListJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, rulesListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetTransactionRules" {
			t.Fatalf("operation = %q, want GetTransactionRules", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"transactionRules":[
			{"id":"rule-1","order":1,"merchantCriteriaUseOriginalStatement":false,"merchantCriteria":[],"merchantNameCriteria":[{"operator":"contains","value":"Uber"}],"amountCriteria":null,"categoryIds":[],"accountIds":[],"setCategoryAction":{"id":"cat-1","name":"Transportation"},"setMerchantAction":null,"addTagsAction":[],"setHideFromReportsAction":null,"reviewStatusAction":null,"recentApplicationCount":5,"lastAppliedAt":"2026-05-01"},
			{"id":"rule-2","order":2,"merchantCriteriaUseOriginalStatement":false,"merchantCriteria":[],"merchantNameCriteria":[{"operator":"eq","value":"Netflix"}],"amountCriteria":{"operator":"eq","isExpense":true,"value":15.99},"categoryIds":["cat-2"],"accountIds":[],"setCategoryAction":{"id":"cat-2","name":"Entertainment"},"setMerchantAction":null,"addTagsAction":[],"setHideFromReportsAction":null,"reviewStatusAction":null,"recentApplicationCount":12,"lastAppliedAt":"2026-05-15"}
		]}}`), nil
	})

	out := captureStdout(t, func() {
		rulesListCmd.Run(rulesListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"rules.list"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"set_category_action"`) {
		t.Fatalf("output missing set_category_action = %q", out)
	}
	if !strings.Contains(out, "Transportation") {
		t.Fatalf("output missing Transportation = %q", out)
	}
	if !strings.Contains(out, "Uber") {
		t.Fatalf("output missing Uber = %q", out)
	}
	if !strings.Contains(out, "contains") {
		t.Fatalf("output missing operator = %q", out)
	}
}

func testRulesUpdateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, rulesUpdateCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_UpdateTransactionRuleMutationV2" {
			t.Fatalf("operation = %q, want Common_UpdateTransactionRuleMutationV2", gqlReq.OperationName)
		}
		input := gqlReq.Variables["input"].(map[string]any)
		if input["id"] != "rule-1" {
			t.Fatalf("input id = %v, want rule-1", input["id"])
		}
		return testutil.JSONResponse(`{"data":{"updateTransactionRuleV2":{}}}`), nil
	})

	ruleMerchantOperator = ""
	ruleMerchantValue = ""
	ruleSetCategoryID = ""
	_ = rulesUpdateCmd.Flags().Set("merchant-operator", "contains")
	_ = rulesUpdateCmd.Flags().Set("merchant-value", "Lyft")
	_ = rulesUpdateCmd.Flags().Set("set-category-id", "cat-transport")
	out := captureStdout(t, func() {
		rulesUpdateCmd.Run(rulesUpdateCmd, []string{"rule-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"rules.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"status":"updated"`) {
		t.Fatalf("output missing status = %q", out)
	}
}

func testRulesDeleteJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, rulesDeleteCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_DeleteTransactionRule" {
			t.Fatalf("operation = %q, want Common_DeleteTransactionRule", gqlReq.OperationName)
		}
		if gqlReq.Variables["id"] != "rule-old" {
			t.Fatalf("variables id = %v, want rule-old", gqlReq.Variables["id"])
		}
		return testutil.JSONResponse(`{"data":{"deleteTransactionRule":{"deleted":true,"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		rulesDeleteCmd.Run(rulesDeleteCmd, []string{"rule-old"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"rules.delete"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"status":"deleted"`) {
		t.Fatalf("output missing status = %q", out)
	}
}
