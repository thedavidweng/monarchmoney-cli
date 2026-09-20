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
	t.Run("show", testTagsShowJSON)
	t.Run("update", testTagsUpdateJSON)
	t.Run("delete", testTagsDeleteJSON)
	t.Run("reorder", testTagsReorderJSON)
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
		if gqlReq.OperationName != "Common_GetHouseholdTransactionTags" {
			t.Fatalf("operation = %q, want Common_GetHouseholdTransactionTags", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"householdTransactionTags":[
			{"id":"tag-1","name":"reimbursable","color":"#ff0000","order":1},
			{"id":"tag-2","name":"tax-deductible","color":"#00ff00","order":2}
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
		if gqlReq.OperationName != "Common_CreateTransactionTag" {
			t.Fatalf("operation = %q, want Common_CreateTransactionTag", gqlReq.OperationName)
		}
		input, _ := gqlReq.Variables["input"].(map[string]any)
		if input["name"] != "vacation" {
			t.Fatalf("variables input.name = %v, want vacation", input["name"])
		}
		if input["color"] != "#0000ff" {
			t.Fatalf("variables input.color = %v, want #0000ff", input["color"])
		}
		return testutil.JSONResponse(`{"data":{"createTransactionTag":{"tag":{"id":"tag-new","name":"vacation","color":"#0000ff","order":3},"errors":null}}}`), nil
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

func testTagsShowJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, tagsShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_GetHouseholdTransactionTags" {
			t.Fatalf("operation = %q, want Common_GetHouseholdTransactionTags", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"householdTransactionTags":[{"id":"tag-1","name":"reimbursable","color":"#ff0000","order":1}]}}`), nil
	})

	out := captureStdout(t, func() {
		tagsShowCmd.Run(tagsShowCmd, []string{"tag-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"tags.show"`) || !strings.Contains(out, "reimbursable") {
		t.Fatalf("output = %q", out)
	}
}

func testTagsUpdateJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, tagsUpdateCmd)
	saveTestSession(t, sessionPath)

	_ = tagsUpdateCmd.Flags().Set("name", "reimbursed")
	t.Cleanup(func() { _ = tagsUpdateCmd.Flags().Set("name", "") })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_GetHouseholdTransactionTags":
			return testutil.JSONResponse(`{"data":{"householdTransactionTags":[{"id":"tag-1","name":"reimbursable","color":"#ff0000","order":1}]}}`), nil
		case "Common_UpdateTransactionTag":
			input, _ := gqlReq.Variables["input"].(map[string]any)
			if input["id"] != "tag-1" || input["name"] != "reimbursed" || input["color"] != "#ff0000" {
				t.Fatalf("update input = %v", input)
			}
			return testutil.JSONResponse(`{"data":{"updateTransactionTag":{"tag":{"id":"tag-1","name":"reimbursed","color":"#ff0000","order":1},"errors":null}}}`), nil
		default:
			t.Fatalf("operation = %q, want tag update ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		tagsUpdateCmd.Run(tagsUpdateCmd, []string{"tag-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"tags.update"`) || !strings.Contains(out, "reimbursed") {
		t.Fatalf("output = %q", out)
	}
}

func testTagsDeleteJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, tagsDeleteCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_DeleteHouseholdTransactionTag" {
			t.Fatalf("operation = %q, want Common_DeleteHouseholdTransactionTag", gqlReq.OperationName)
		}
		if gqlReq.Variables["tagId"] != "tag-1" {
			t.Fatalf("variables tagId = %v", gqlReq.Variables["tagId"])
		}
		return testutil.JSONResponse(`{"data":{"deleteTransactionTag":{"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		tagsDeleteCmd.Run(tagsDeleteCmd, []string{"tag-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"tags.delete"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testTagsReorderJSON(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, tagsReorderCmd)
	saveTestSession(t, sessionPath)

	_ = tagsReorderCmd.Flags().Set("order", "2")

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_UpdateTransactionTagOrder" {
			t.Fatalf("operation = %q, want Common_UpdateTransactionTagOrder", gqlReq.OperationName)
		}
		if gqlReq.Variables["tagId"] != "tag-1" || gqlReq.Variables["order"] != float64(2) {
			t.Fatalf("reorder variables = %v", gqlReq.Variables)
		}
		return testutil.JSONResponse(`{"data":{"updateTransactionTagOrder":{"householdTransactionTags":[{"id":"tag-1","name":"reimbursable","color":"#ff0000","order":2}]}}}`), nil
	})

	out := captureStdout(t, func() {
		tagsReorderCmd.Run(tagsReorderCmd, []string{"tag-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"tags.reorder"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func TestTagsHumanOutput(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath,
		tagsListCmd, tagsShowCmd, tagsCreateCmd, tagsUpdateCmd, tagsDeleteCmd, tagsReorderCmd)
	saveTestSession(t, sessionPath)

	oldJSON := jsonMode
	jsonMode = false
	t.Cleanup(func() { jsonMode = oldJSON })

	_ = tagsCreateCmd.Flags().Set("name", "vacation")
	_ = tagsCreateCmd.Flags().Set("color", "#0000ff")
	_ = tagsUpdateCmd.Flags().Set("name", "reimbursed")
	_ = tagsReorderCmd.Flags().Set("order", "2")
	t.Cleanup(func() {
		_ = tagsCreateCmd.Flags().Set("name", "")
		_ = tagsCreateCmd.Flags().Set("color", "#000000")
		_ = tagsUpdateCmd.Flags().Set("name", "")
		_ = tagsReorderCmd.Flags().Set("order", "0")
	})

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_GetHouseholdTransactionTags":
			return testutil.JSONResponse(`{"data":{"householdTransactionTags":[{"id":"tag-1","name":"reimbursable","color":"#ff0000","order":1}]}}`), nil
		case "Common_CreateTransactionTag":
			return testutil.JSONResponse(`{"data":{"createTransactionTag":{"tag":{"id":"tag-new","name":"vacation","color":"#0000ff","order":3},"errors":null}}}`), nil
		case "Common_UpdateTransactionTag":
			return testutil.JSONResponse(`{"data":{"updateTransactionTag":{"tag":{"id":"tag-1","name":"reimbursed","color":"#ff0000","order":1},"errors":null}}}`), nil
		case "Common_DeleteHouseholdTransactionTag":
			return testutil.JSONResponse(`{"data":{"deleteTransactionTag":{"errors":null}}}`), nil
		case "Common_UpdateTransactionTagOrder":
			return testutil.JSONResponse(`{"data":{"updateTransactionTagOrder":{"householdTransactionTags":[{"id":"tag-1","name":"reimbursable","color":"#ff0000","order":2}]}}}`), nil
		default:
			t.Fatalf("operation = %q", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		tagsListCmd.Run(tagsListCmd, nil)
		tagsShowCmd.Run(tagsShowCmd, []string{"tag-1"})
		tagsCreateCmd.Run(tagsCreateCmd, nil)
		tagsUpdateCmd.Run(tagsUpdateCmd, []string{"tag-1"})
		tagsDeleteCmd.Run(tagsDeleteCmd, []string{"tag-1"})
		tagsReorderCmd.Run(tagsReorderCmd, []string{"tag-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	for _, want := range []string{"reimbursable", "Successfully created tag", "Successfully updated tag", "Successfully deleted tag", "Successfully moved tag"} {
		if !strings.Contains(out, want) {
			t.Fatalf("human output missing %q in %q", want, out)
		}
	}
}
