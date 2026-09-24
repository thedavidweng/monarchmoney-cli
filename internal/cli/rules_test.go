package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestRules(t *testing.T) {
	t.Run("list", testRulesListJSON)
	t.Run("update", testRulesUpdateJSON)
	t.Run("delete", testRulesDeleteJSON)
	t.Run("reorder", testRulesReorderJSON)
	t.Run("reorder_negative", testRulesReorderNegativeOrder)
	t.Run("reorder_api_error", testRulesReorderAPIError)
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
		input, _ := gqlReq.Variables["input"].(map[string]any)
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

func testRulesReorderJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, rulesReorderCmd)
	saveTestSession(t, sessionPath)

	_ = rulesReorderCmd.Flags().Set("order", "0")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "GetTransactionRules":
			return testutil.JSONResponse(`{"data":{"transactionRules":[{"id":"rule-1","order":1}]}}`), nil
		case "Web_UpdateRuleOrderMutation":
			if gqlReq.Variables["id"] != "rule-1" {
				t.Fatalf("variables = %#v, want id rule-1", gqlReq.Variables)
			}
			return testutil.JSONResponse(`{"data":{"updateTransactionRuleOrderV2":{"transactionRules":[{"id":"rule-1","order":0}]}}}`), nil
		default:
			t.Fatalf("operation = %q, want rules operations", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		rulesReorderCmd.Run(rulesReorderCmd, []string{"rule-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"rules.reorder"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testRulesReorderNegativeOrder(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, rulesReorderCmd)
	saveTestSession(t, sessionPath)

	_ = rulesReorderCmd.Flags().Set("order", "-1")

	out := captureStdout(t, func() {
		rulesReorderCmd.Run(rulesReorderCmd, []string{"rule-1"})
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "--order must be 0 or greater") {
		t.Fatalf("output = %q, want order guidance", out)
	}
	_ = rulesReorderCmd.Flags().Set("order", "0")
}

func TestRulesHumanOutputGap(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, rulesReorderCmd)
	saveTestSession(t, sessionPath)

	oldJSON := jsonMode
	jsonMode = false
	t.Cleanup(func() { jsonMode = oldJSON })

	_ = rulesReorderCmd.Flags().Set("order", "0")
	t.Cleanup(func() { _ = rulesReorderCmd.Flags().Set("order", "0") })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "GetTransactionRules":
			return testutil.JSONResponse(`{"data":{"transactionRules":[{"id":"rule-1","order":1}]}}`), nil
		case "Web_UpdateRuleOrderMutation":
			return testutil.JSONResponse(`{"data":{"updateTransactionRuleOrderV2":{"transactionRules":[{"id":"rule-1","order":0}]}}}`), nil
		default:
			t.Fatalf("operation = %q", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		rulesReorderCmd.Run(rulesReorderCmd, []string{"rule-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, "Moved rule rule-1 from 1 to 0.") {
		t.Fatalf("human output missing move confirmation in %q", out)
	}
}

func testRulesReorderAPIError(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, rulesReorderCmd)
	saveTestSession(t, sessionPath)

	_ = rulesReorderCmd.Flags().Set("order", "0")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName == "GetTransactionRules" {
			return testutil.JSONResponse(`{"data":{"transactionRules":[{"id":"rule-1","order":1}]}}`), nil
		}
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	})

	out := captureStdout(t, func() {
		rulesReorderCmd.Run(rulesReorderCmd, []string{"rule-1"})
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want API failure; output=%q", out)
	}
}

func resetRuleFlags(t *testing.T) {
	t.Helper()
	ruleMerchantOperator = ""
	ruleMerchantValue = ""
	ruleLegacyMerchantOperator = ""
	ruleLegacyMerchantValue = ""
	ruleUseOriginalStatement = false
	ruleOriginalStatementOperator = ""
	ruleOriginalStatementValue = ""
	ruleAmountOperator = ""
	ruleAmountValue = 0
	ruleAmountValueUpper = 0
	ruleAmountIsExpense = true
	ruleCategoryIDs = nil
	ruleSetCategoryID = ""
	ruleSetMerchant = ""
	ruleAddTagIDs = nil
	ruleAccountIDs = nil
	ruleCriteriaOwnerUserIDs = nil
	ruleCriteriaOwnerIsJoint = false
	ruleCriteriaBusinessEntityIDs = nil
	ruleCriteriaBusinessEntityIsUnassigned = false
	ruleHideFromReports = false
	ruleReviewStatus = ""
	ruleNeedsReviewByUserID = ""
	ruleLinkGoalID = ""
	ruleLinkSavingsGoalID = ""
	ruleLinkToPaydownBudget = false
	ruleSendNotification = false
	ruleActionSetOwner = ""
	ruleActionSetOwnerIsJoint = false
	ruleActionSetBusinessEntity = ""
	ruleActionSetBusinessEntityIsUnassigned = false
	ruleSplitFile = ""
	ruleApplyToExisting = false
	for _, cmd := range []*cobra.Command{rulesCreateCmd, rulesUpdateCmd} {
		cmd.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })
	}
}

func setFullRuleFlags(t *testing.T, cmd *cobra.Command, splitPath string) {
	t.Helper()
	mustSet := func(name, value string) {
		t.Helper()
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("Set(%q) error = %v", name, err)
		}
	}
	mustSet("merchant-operator", "contains")
	mustSet("merchant-value", "Uber")
	mustSet("merchant-criteria-operator", "eq")
	mustSet("merchant-criteria-value", "Legacy Co")
	mustSet("use-original-statement", "true")
	mustSet("original-statement-operator", "contains")
	mustSet("original-statement-value", "UBER *TRIP")
	mustSet("amount-operator", "between")
	mustSet("amount-value", "10")
	mustSet("amount-value-upper", "50")
	mustSet("category-id", "cat-1")
	mustSet("set-category-id", "cat-9")
	mustSet("set-merchant", "Uber Rides")
	mustSet("add-tag-id", "tag-1")
	mustSet("account-id", "acc-1")
	mustSet("criteria-owner-user-id", "u-1")
	mustSet("criteria-owner-joint", "true")
	mustSet("criteria-business-entity-id", "b-1")
	mustSet("criteria-business-entity-unassigned", "true")
	mustSet("hide-from-reports", "true")
	mustSet("review-status", "needs_review")
	mustSet("needs-review-by-user-id", "u-2")
	mustSet("link-goal-id", "goal-1")
	mustSet("link-savings-goal-id", "sgoal-1")
	mustSet("link-to-paydown-budget", "true")
	mustSet("send-notification", "true")
	mustSet("action-set-owner", "u-3")
	mustSet("action-set-owner-joint", "true")
	mustSet("action-set-business-entity", "b-2")
	mustSet("action-set-business-entity-unassigned", "true")
	mustSet("split-file", splitPath)
	mustSet("apply-to-existing", "true")
}

func wantFullRuleInput(extra map[string]any) map[string]any {
	input := map[string]any{
		"merchantNameCriteria":                 []any{map[string]any{"operator": "contains", "value": "Uber"}},
		"merchantCriteria":                     []any{map[string]any{"operator": "eq", "value": "Legacy Co"}},
		"merchantCriteriaUseOriginalStatement": true,
		"originalStatementCriteria":            []any{map[string]any{"operator": "contains", "value": "UBER *TRIP"}},
		"amountCriteria":                       map[string]any{"operator": "between", "isExpense": true, "value": 10.0, "valueRange": map[string]any{"lower": 10.0, "upper": 50.0}},
		"categoryIds":                          []any{"cat-1"},
		"accountIds":                           []any{"acc-1"},
		"criteriaOwnerUserIds":                 []any{"u-1"},
		"criteriaOwnerIsJoint":                 true,
		"criteriaBusinessEntityIds":            []any{"b-1"},
		"criteriaBusinessEntityIsUnassigned":   true,
		"setCategoryAction":                    "cat-9",
		"setMerchantAction":                    "Uber Rides",
		"addTagsAction":                        []any{"tag-1"},
		"setHideFromReportsAction":             true,
		"reviewStatusAction":                   "needs_review",
		"needsReviewByUserAction":              "u-2",
		"linkGoalAction":                       "goal-1",
		"linkSavingsGoalAction":                "sgoal-1",
		"setLinkToPaydownBudgetAction":         true,
		"sendNotificationAction":               true,
		"actionSetOwner":                       "u-3",
		"actionSetOwnerIsJoint":                true,
		"actionSetBusinessEntity":              "b-2",
		"actionSetBusinessEntityIsUnassigned":  true,
		"splitTransactionsAction":              map[string]any{"amountType": "PERCENTAGE", "splitsInfo": []any{map[string]any{"amount": 0.6, "categoryId": "cat-1"}}},
		"applyToExistingTransactions":          true,
	}
	for k, v := range extra {
		input[k] = v
	}
	return map[string]any{"input": input}
}

func TestRulesCreateFullParity(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, rulesCreateCmd)
	saveTestSession(t, sessionPath)
	t.Cleanup(func() { resetRuleFlags(t) })

	splitPath := filepath.Join(dir, "split.json")
	if err := os.WriteFile(splitPath, []byte(`{"amountType":"PERCENTAGE","splitsInfo":[{"amount":0.6,"categoryId":"cat-1"}]}`), 0o600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	setFullRuleFlags(t, rulesCreateCmd, splitPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_CreateTransactionRuleMutationV2" {
			t.Fatalf("operation = %q, want Common_CreateTransactionRuleMutationV2", gqlReq.OperationName)
		}
		if !reflect.DeepEqual(gqlReq.Variables, wantFullRuleInput(nil)) {
			got, _ := json.Marshal(gqlReq.Variables)
			t.Fatalf("variables = %s", got)
		}
		return testutil.JSONResponse(`{"data":{"createTransactionRuleV2":{}}}`), nil
	})

	out := captureStdout(t, func() {
		rulesCreateCmd.Run(rulesCreateCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"status":"created"`) {
		t.Fatalf("output missing status = %q", out)
	}
}

func TestRulesUpdateFullParity(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, rulesUpdateCmd)
	saveTestSession(t, sessionPath)
	t.Cleanup(func() { resetRuleFlags(t) })

	splitPath := filepath.Join(dir, "split.json")
	if err := os.WriteFile(splitPath, []byte(`{"amountType":"PERCENTAGE","splitsInfo":[{"amount":0.6,"categoryId":"cat-1"}]}`), 0o600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	setFullRuleFlags(t, rulesUpdateCmd, splitPath)

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
		if !reflect.DeepEqual(gqlReq.Variables, wantFullRuleInput(map[string]any{"id": "rule-1"})) {
			got, _ := json.Marshal(gqlReq.Variables)
			t.Fatalf("variables = %s", got)
		}
		return testutil.JSONResponse(`{"data":{"updateTransactionRuleV2":{}}}`), nil
	})

	out := captureStdout(t, func() {
		rulesUpdateCmd.Run(rulesUpdateCmd, []string{"rule-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"status":"updated"`) {
		t.Fatalf("output missing status = %q", out)
	}
}

func TestRulesCreateValidation(t *testing.T) {
	cases := []struct {
		name  string
		setup func()
		want  string
	}{
		{"bad_review_status", func() { _ = rulesCreateCmd.Flags().Set("review-status", "bogus") }, "--review-status must be needs_review or reviewed"},
		{"between_without_upper", func() { _ = rulesCreateCmd.Flags().Set("amount-operator", "between") }, "--amount-value-upper is required when --amount-operator=between"},
		{"upper_without_between", func() {
			_ = rulesCreateCmd.Flags().Set("amount-operator", "gt")
			_ = rulesCreateCmd.Flags().Set("amount-value", "10")
			_ = rulesCreateCmd.Flags().Set("amount-value-upper", "50")
		}, "--amount-value-upper requires --amount-operator=between"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			sessionPath := filepath.Join(dir, "session.json")
			exitCode := withWriteCommandTestDefaults(t, sessionPath, rulesCreateCmd)
			saveTestSession(t, sessionPath)
			t.Cleanup(func() { resetRuleFlags(t) })
			tc.setup()

			out := captureStdout(t, func() {
				rulesCreateCmd.Run(rulesCreateCmd, nil)
			})

			if *exitCode == 0 {
				t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
			}
			if !strings.Contains(out, tc.want) {
				t.Fatalf("output = %q, want %q", out, tc.want)
			}
		})
	}
}

func TestRulesCreateSplitFileValidation(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad-split.json")
	if err := os.WriteFile(badPath, []byte(`{"amountType":"WRONG","splitsInfo":[{"amount":1}]}`), 0o600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, rulesCreateCmd)
	saveTestSession(t, sessionPath)
	t.Cleanup(func() { resetRuleFlags(t) })
	_ = rulesCreateCmd.Flags().Set("split-file", badPath)

	out := captureStdout(t, func() {
		rulesCreateCmd.Run(rulesCreateCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "split amountType must be ABSOLUTE or PERCENTAGE") {
		t.Fatalf("output = %q", out)
	}
}

func TestRulesListFullParity(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, rulesListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return testutil.JSONResponse(`{"data":{"transactionRules":[
			{"id":"rule-1","order":1,"merchantCriteriaUseOriginalStatement":true,"merchantCriteria":[],"merchantNameCriteria":[{"operator":"contains","value":"Uber"}],"originalStatementCriteria":[{"operator":"contains","value":"UBER"}],"amountCriteria":null,"categoryIds":[],"accountIds":[],"setCategoryAction":{"id":"cat-1","name":"Transportation"},"setMerchantAction":{"id":"m-9","name":"Uber Rides"},"addTagsAction":[],"setHideFromReportsAction":true,"reviewStatusAction":"needs_review","splitTransactionsAction":{"amountType":"PERCENTAGE","splitsInfo":[{"amount":0.6}]},"recentApplicationCount":5,"lastAppliedAt":"2026-05-01"}
		]}}`), nil
	})

	out := captureStdout(t, func() {
		rulesListCmd.Run(rulesListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	for _, want := range []string{`"original_statement_criteria"`, `"set_merchant_action"`, `"review_status_action"`, `"split_transactions_action"`, `"set_hide_from_reports_action"`, "Uber Rides"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q in %q", want, out)
		}
	}
}
