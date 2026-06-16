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
	t.Run("delete", testCategoriesDeleteJSON)
	t.Run("delete_many", testCategoriesDeleteManyJSON)
	t.Run("delete_many_missing_file", testCategoriesDeleteManyMissingFile)
	t.Run("delete_many_file_not_found", testCategoriesDeleteManyFileNotFound)
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
		if gqlReq.OperationName != "GetCategoryGroups" {
			t.Fatalf("operation = %q, want GetCategoryGroups", gqlReq.OperationName)
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
		if gqlReq.OperationName != "DeleteCategory" {
			t.Fatalf("operation = %q, want DeleteCategory", gqlReq.OperationName)
		}
		if gqlReq.Variables["id"] != "cat-old" {
			t.Fatalf("variables id = %v, want cat-old", gqlReq.Variables["id"])
		}
		return testutil.JSONResponse(`{"data":{"deleteCategory":{"ok":true}}}`), nil
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
	if err := os.WriteFile(idsFile, []byte("cat-1\ncat-2\ncat-3\n"), 0600); err != nil {
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
		ids := gqlReq.Variables["ids"].([]any)
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
