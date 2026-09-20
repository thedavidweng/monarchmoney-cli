package cli

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestCategories(t *testing.T) {
	t.Run("list", testCategoriesListWithGroups)
	t.Run("groups", testCategoriesGroupsJSON)
	t.Run("show", testCategoriesShowJSON)
	t.Run("delete", testCategoriesDeleteJSON)
	t.Run("delete_many", testCategoriesDeleteManyJSON)
	t.Run("delete_many_missing_file", testCategoriesDeleteManyMissingFile)
	t.Run("delete_many_file_not_found", testCategoriesDeleteManyFileNotFound)
	t.Run("update", testCategoriesUpdateJSON)
	t.Run("reactivate", testCategoriesReactivateJSON)
	t.Run("reorder", testCategoriesReorderJSON)
	t.Run("rollover", testCategoriesRolloverJSON)
	t.Run("group_create", testCategoriesGroupCreateJSON)
	t.Run("group_update", testCategoriesGroupUpdateJSON)
	t.Run("group_delete", testCategoriesGroupDeleteJSON)
	t.Run("group_reorder", testCategoriesGroupReorderJSON)
}

func testCategoriesListWithGroups(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, categoriesListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetCategories" {
			t.Fatalf("operation = %q, want GetCategories", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"categories":[
			{"id":"cat-1","name":"Dining","order":1,"icon":"utensils","group":{"id":"grp-1","name":"Food & Drink","type":"expense"}},
			{"id":"cat-2","name":"Income","order":2,"icon":"dollar","group":{"id":"grp-2","name":"Income","type":"income"}}
		]}}`), nil
	})

	out := captureStdout(t, func() {
		categoriesListCmd.Run(categoriesListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.list"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"name":"Dining"`) {
		t.Fatalf("output missing Dining = %q", out)
	}
	if !strings.Contains(out, `"group_name":"Food`) {
		t.Fatalf("output missing group name = %q", out)
	}
}

func testCategoriesGroupsJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, categoriesGroupsCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "ManageGetCategoryGroups" {
			t.Fatalf("operation = %q, want ManageGetCategoryGroups", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"categoryGroups":[
			{"id":"grp-1","name":"Food & Drink","type":"expense","categories":[{"id":"cat-1","name":"Dining"}]},
			{"id":"grp-2","name":"Income","type":"income","categories":[]}
		]}}`), nil
	})

	out := captureStdout(t, func() {
		categoriesGroupsCmd.Run(categoriesGroupsCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.groups"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"name":"Food`) {
		t.Fatalf("output missing group name = %q", out)
	}
	if !strings.Contains(out, `"type":"expense"`) {
		t.Fatalf("output missing group type = %q", out)
	}
}

func testCategoriesDeleteJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, categoriesDeleteCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_DeleteCategory" {
			t.Fatalf("operation = %q, want Web_DeleteCategory", gqlReq.OperationName)
		}
		if gqlReq.Variables["id"] != "cat-old" {
			t.Fatalf("variables id = %v, want cat-old", gqlReq.Variables["id"])
		}
		return testutil.JSONResponse(`{"data":{"deleteCategory":{"errors":null,"deleted":true}}}`), nil
	})

	out := captureStdout(t, func() {
		categoriesDeleteCmd.Run(categoriesDeleteCmd, []string{"cat-old"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.delete"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"status":"deleted"`) {
		t.Fatalf("output missing status = %q", out)
	}
}

func testCategoriesDeleteManyJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, categoriesDeleteManyCmd)
	saveTestSession(t, sessionPath)

	idsFile := filepath.Join(dir, "ids.txt")
	if err := os.WriteFile(idsFile, []byte("cat-1\ncat-2\ncat-3\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "DeleteCategories" {
			t.Fatalf("operation = %q, want DeleteCategories", gqlReq.OperationName)
		}
		ids, _ := gqlReq.Variables["ids"].([]any)
		if len(ids) != 3 {
			t.Fatalf("variables ids count = %d, want 3", len(ids))
		}
		return testutil.JSONResponse(`{"data":{"deleteTransactionCategories":{"ok":true}}}`), nil
	})

	categoryFile = ""
	_ = categoriesDeleteManyCmd.Flags().Set("file", idsFile)
	out := captureStdout(t, func() {
		categoriesDeleteManyCmd.Run(categoriesDeleteManyCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.delete-many"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"status":"categories deleted"`) {
		t.Fatalf("output missing status = %q", out)
	}
}

func testCategoriesDeleteManyMissingFile(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, categoriesDeleteManyCmd)
	saveTestSession(t, sessionPath)

	categoryFile = ""
	out := captureStdout(t, func() {
		categoriesDeleteManyCmd.Run(categoriesDeleteManyCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "--file is required") {
		t.Fatalf("output = %q, want --file required message", out)
	}
}

func testCategoriesDeleteManyFileNotFound(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, categoriesDeleteManyCmd)
	saveTestSession(t, sessionPath)

	categoryFile = ""
	_ = categoriesDeleteManyCmd.Flags().Set("file", filepath.Join(dir, "nonexistent.txt"))
	out := captureStdout(t, func() {
		categoriesDeleteManyCmd.Run(categoriesDeleteManyCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want file error; output=%q", out)
	}
	if !strings.Contains(out, "failed to open file") {
		t.Fatalf("output = %q, want file open failure", out)
	}
}

func testCategoriesUpdateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, categoriesUpdateCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_UpdateCategory" {
			t.Fatalf("operation = %q, want Web_UpdateCategory", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"updateCategory":{"errors":[],"category":{"id":"cat-1","name":"Groceries","icon":"cart","budgetVariability":"flexible","excludeFromBudget":false,"group":{"id":"grp-1","type":"expense"}}}}}`), nil
	})

	_ = categoriesUpdateCmd.Flags().Set("name", "Groceries")
	out := captureStdout(t, func() {
		categoriesUpdateCmd.Run(categoriesUpdateCmd, []string{"cat-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"name":"Groceries"`) {
		t.Fatalf("output missing name = %q", out)
	}
}

func testCategoriesRolloverJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, categoriesRolloverCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetCategoryRollover" {
			t.Fatalf("operation = %q, want GetCategoryRollover", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"category":{"id":"cat-1","name":"Food","rolloverPeriod":{"id":"rp1","startMonth":"2026-01-01","startingBalance":100,"type":"monthly","frequency":"monthly","targetAmount":500}}}}`), nil
	})

	out := captureStdout(t, func() {
		categoriesRolloverCmd.Run(categoriesRolloverCmd, []string{"cat-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.rollover"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"name":"Food"`) {
		t.Fatalf("output missing name = %q", out)
	}
	if !strings.Contains(out, `"start_month":"2026-01-01"`) {
		t.Fatalf("output missing start_month = %q", out)
	}
}

func testCategoriesGroupUpdateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, categoriesGroupUpdateCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_UpdateCategoryGroup" {
			t.Fatalf("operation = %q, want Common_UpdateCategoryGroup", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"updateCategoryGroup":{"categoryGroup":{"id":"grp-1","name":"Food & Drink","order":1,"type":"expense","color":"","groupLevelBudgetingEnabled":false,"budgetVariability":"fixed"}}}}`), nil
	})

	_ = categoriesGroupUpdateCmd.Flags().Set("name", "Food & Drink")
	out := captureStdout(t, func() {
		categoriesGroupUpdateCmd.Run(categoriesGroupUpdateCmd, []string{"grp-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.groups.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"name":"Food \u0026 Drink"`) {
		t.Fatalf("output missing name = %q", out)
	}
}

func testCategoriesShowJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, categoriesShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_GetEditCategory" {
			t.Fatalf("operation = %q, want Web_GetEditCategory", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"category":{"id":"cat-1","order":1,"name":"Dining","icon":"utensils","group":{"id":"grp-1","name":"Food & Drink","type":"expense"}}}}`), nil
	})

	out := captureStdout(t, func() {
		categoriesShowCmd.Run(categoriesShowCmd, []string{"cat-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.show"`) || !strings.Contains(out, "Dining") {
		t.Fatalf("output = %q", out)
	}
}

func testCategoriesReactivateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, categoriesReactivateCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_RestoreCategory" {
			t.Fatalf("operation = %q, want Web_RestoreCategory", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"restoreCategory":{"errors":null,"category":{"id":"cat-1","order":1,"name":"Dining","icon":"utensils","group":{"id":"grp-1","name":"Food","type":"expense"}}}}}`), nil
	})

	out := captureStdout(t, func() {
		categoriesReactivateCmd.Run(categoriesReactivateCmd, []string{"cat-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.reactivate"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testCategoriesReorderJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, categoriesReorderCmd)
	saveTestSession(t, sessionPath)

	_ = categoriesReorderCmd.Flags().Set("group", "grp-1")
	_ = categoriesReorderCmd.Flags().Set("order", "3")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_UpdateCategoryOrder" {
			t.Fatalf("operation = %q, want Web_UpdateCategoryOrder", gqlReq.OperationName)
		}
		if gqlReq.Variables["categoryGroupId"] != "grp-1" || gqlReq.Variables["order"] != float64(3) {
			t.Fatalf("reorder variables = %v", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"updateCategoryOrderInCategoryGroup":{"category":{"id":"cat-1","order":3,"name":"Dining","icon":"utensils","group":{"id":"grp-1","name":"Food","type":"expense"}}}}}`), nil
	})

	out := captureStdout(t, func() {
		categoriesReorderCmd.Run(categoriesReorderCmd, []string{"cat-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.reorder"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testCategoriesGroupCreateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, categoriesGroupCreateCmd)
	saveTestSession(t, sessionPath)

	_ = categoriesGroupCreateCmd.Flags().Set("name", "Pets")
	_ = categoriesGroupCreateCmd.Flags().Set("type", "expense")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_CreateCategoryGroup" {
			t.Fatalf("operation = %q, want Common_CreateCategoryGroup", gqlReq.OperationName)
		}
		input, _ := gqlReq.Variables["input"].(map[string]any)
		if input["name"] != "Pets" || input["type"] != "expense" {
			t.Fatalf("create group input = %v", input)
		}
		return testutil.JSONResponse(`{"data":{"createCategoryGroup":{"categoryGroup":{"id":"grp-9","name":"Pets","order":9,"type":"expense"}}}}`), nil
	})

	out := captureStdout(t, func() {
		categoriesGroupCreateCmd.Run(categoriesGroupCreateCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.groups.create"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testCategoriesGroupDeleteJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, categoriesGroupDeleteCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_DeleteCategoryGroup" {
			t.Fatalf("operation = %q, want Common_DeleteCategoryGroup", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"deleteCategoryGroup":{"deleted":true,"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		categoriesGroupDeleteCmd.Run(categoriesGroupDeleteCmd, []string{"grp-9"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.groups.delete"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testCategoriesGroupReorderJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, categoriesGroupReorderCmd)
	saveTestSession(t, sessionPath)

	_ = categoriesGroupReorderCmd.Flags().Set("order", "1")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Web_UpdateCategoryGroupOrder" {
			t.Fatalf("operation = %q, want Web_UpdateCategoryGroupOrder", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"updateCategoryGroupOrder":{"categoryGroups":[{"id":"grp-9","name":"Pets","order":1,"type":"expense"}]}}}`), nil
	})

	out := captureStdout(t, func() {
		categoriesGroupReorderCmd.Run(categoriesGroupReorderCmd, []string{"grp-9"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"categories.groups.reorder"`) {
		t.Fatalf("output missing command = %q", out)
	}
}
