package cli

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestHousehold(t *testing.T) {
	t.Run("show", testHouseholdShow)
	t.Run("members", testHouseholdMembers)
	t.Run("member", testHouseholdMember)
	t.Run("me", testHouseholdMe)
	t.Run("me_update", testHouseholdMeUpdate)
	t.Run("preferences", testHouseholdPreferences)
	t.Run("preferences_update", testHouseholdPreferencesUpdate)
}

func testHouseholdShow(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, householdShowCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_GetMyHousehold" {
			t.Fatalf("operation = %q, want Common_GetMyHousehold", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"myHousehold":{"id":"hh-1","name":"Smiths","address":"1 Main St","city":"SF","state":"CA","zipCode":"94101","country":"US"}}}`), nil
	})

	out := captureStdout(t, func() {
		householdShowCmd.Run(householdShowCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"household.show"`) || !strings.Contains(out, "Smiths") {
		t.Fatalf("output = %q", out)
	}
}

func testHouseholdMembers(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, householdMembersCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_GetHouseholdMembers" {
			t.Fatalf("operation = %q, want Common_GetHouseholdMembers", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"myHousehold":{"id":"hh-1","users":[{"id":"u-1","name":"Ann","displayName":"Ann S","email":"ann@example.com","householdRole":"owner","hasMfaOn":true,"profilePictureUrl":""}]}}}`), nil
	})

	out := captureStdout(t, func() {
		householdMembersCmd.Run(householdMembersCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"household.members"`) || !strings.Contains(out, "ann@example.com") {
		t.Fatalf("output = %q", out)
	}
}

func testHouseholdMember(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, householdMemberCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return testutil.JSONResponse(`{"data":{"myHousehold":{"id":"hh-1","users":[{"id":"u-1","name":"Ann","displayName":"Ann S","email":"ann@example.com","householdRole":"owner","hasMfaOn":false,"profilePictureUrl":""}]}}}`), nil
	})

	out := captureStdout(t, func() {
		householdMemberCmd.Run(householdMemberCmd, []string{"u-1"})
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"household.member"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testHouseholdMe(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, householdMeCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_GetMe" {
			t.Fatalf("operation = %q, want Common_GetMe", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"me":{"id":"u-1","email":"ann@example.com","name":"Ann","displayName":"Ann S","timezone":"America/Los_Angeles","householdRole":"owner","hasPassword":true,"hasMfaOn":true,"isSuperuser":false,"profilePictureUrl":"","createdAt":"2024-01-01T00:00:00Z"}}}`), nil
	})

	out := captureStdout(t, func() {
		householdMeCmd.Run(householdMeCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"household.me"`) || !strings.Contains(out, "America/Los_Angeles") {
		t.Fatalf("output = %q", out)
	}
}

func testHouseholdMeUpdate(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, householdMeUpdateCmd)
	saveTestSession(t, sessionPath)

	_ = householdMeUpdateCmd.Flags().Set("timezone", "America/New_York")
	t.Cleanup(func() { _ = householdMeUpdateCmd.Flags().Set("timezone", "") })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_UpdateMe" {
			t.Fatalf("operation = %q, want Common_UpdateMe", gqlReq.OperationName)
		}
		input, _ := gqlReq.Variables["input"].(map[string]any)
		if input["timezone"] != "America/New_York" {
			t.Fatalf("update input = %v", input)
		}
		return testutil.JSONResponse(`{"data":{"updateMe":{"user":{"id":"u-1","email":"ann@example.com","name":"Ann","displayName":"Ann S","timezone":"America/New_York","householdRole":"owner","hasMfaOn":true,"createdAt":"2024-01-01T00:00:00Z"},"errors":null}}}`), nil
	})

	out := captureStdout(t, func() {
		householdMeUpdateCmd.Run(householdMeUpdateCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"household.me.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
}

func testHouseholdPreferences(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withReadCommandTestDefaults(t, sessionPath, householdPreferencesCmd)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName != "Common_GetHouseholdPreferences" {
			t.Fatalf("operation = %q, want Common_GetHouseholdPreferences", gqlReq.OperationName)
		}
		return testutil.JSONResponse(`{"data":{"householdPreferences":{"id":"hp-1","newTransactionsNeedReview":true,"uncategorizedTransactionsNeedReview":false,"pendingTransactionsCanBeEdited":true},"budgetSystem":"flex","budgetApplyToFutureMonthsDefault":true}}`), nil
	})

	out := captureStdout(t, func() {
		householdPreferencesCmd.Run(householdPreferencesCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"household.preferences"`) || !strings.Contains(out, "flex") {
		t.Fatalf("output = %q", out)
	}
}

func testHouseholdPreferencesUpdate(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	exitCode := withWriteCommandTestDefaults(t, sessionPath, householdPreferencesUpdateCmd)
	saveTestSession(t, sessionPath)

	_ = householdPreferencesUpdateCmd.Flags().Set("new-need-review", "true")
	t.Cleanup(func() { _ = householdPreferencesUpdateCmd.Flags().Set("new-need-review", "false") })

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		switch gqlReq.OperationName {
		case "Common_UpdateHouseholdPreferences":
			input, _ := gqlReq.Variables["input"].(map[string]any)
			if input["newTransactionsNeedReview"] != true {
				t.Fatalf("prefs input = %v", input)
			}
			return testutil.JSONResponse(`{"data":{"updateHouseholdPreferences":{"householdPreferences":{"id":"hp-1"}}}}`), nil
		case "Common_GetHouseholdPreferences":
			return testutil.JSONResponse(`{"data":{"householdPreferences":{"id":"hp-1","newTransactionsNeedReview":true},"budgetSystem":"flex"}}`), nil
		default:
			t.Fatalf("operation = %q, want household prefs ops", gqlReq.OperationName)
			return nil, nil
		}
	})

	out := captureStdout(t, func() {
		householdPreferencesUpdateCmd.Run(householdPreferencesUpdateCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !strings.Contains(out, `"command":"household.preferences.update"`) {
		t.Fatalf("output missing command = %q", out)
	}
}
