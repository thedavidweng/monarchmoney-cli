package cli

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestTags(t *testing.T) {
	t.Run("list", testTagsListJSON)
	t.Run("create", testTagsCreateJSON)
}

func testTagsListJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, tagsListCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "GetTags" {
			t.Fatalf("operation = %q, want GetTags", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"householdTransactionTags":[
			{"id":"tag-1","name":"reimbursable","color":"#ff0000"},
			{"id":"tag-2","name":"tax-deductible","color":"#00ff00"}
		]}}`), nil
	})

	out := captureStdout(t, func() {
		tagsListCmd.Run(tagsListCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"tags.list"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, `"name":"reimbursable"`) {
		t.Fatalf("output missing reimbursable = %q", out)
	}
	if !strings.Contains(out, `"color":"#ff0000"`) {
		t.Fatalf("output missing color = %q", out)
	}
}

func testTagsCreateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, tagsCreateCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "CreateTag" {
			t.Fatalf("operation = %q, want CreateTag", gqlReq.OperationName)
		}
		if gqlReq.Variables["name"] != "vacation" {
			t.Fatalf("variables name = %v, want vacation", gqlReq.Variables["name"])
		}
		if gqlReq.Variables["color"] != "#0000ff" {
			t.Fatalf("variables color = %v, want #0000ff", gqlReq.Variables["color"])
		}
		return testutil.JSONResponse(`{"data":{"createHouseholdTransactionTag":{"tag":{"id":"tag-new","name":"vacation","color":"#0000ff"}}}}`), nil
	})

	tagName = ""
	tagColor = ""
	_ = tagsCreateCmd.Flags().Set("name", "vacation")
	_ = tagsCreateCmd.Flags().Set("color", "#0000ff")
	out := captureStdout(t, func() {
		tagsCreateCmd.Run(tagsCreateCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"tags.create"`) {
		t.Fatalf("output missing command = %q", out)
	}
	if !strings.Contains(out, "tag-new") {
		t.Fatalf("output missing tag ID = %q", out)
	}
	if !strings.Contains(out, "vacation") {
		t.Fatalf("output missing tag name = %q", out)
	}
}
