package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestTransactionsFilteringAndMutationSafety(t *testing.T) {
	type request struct {
		OperationName string         `json:"operationName"`
		Variables     map[string]any `json:"variables"`
	}
	requests := make(chan request, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Token test-token" {
			t.Errorf("unexpected request: method=%s authorization=%q", r.Method, r.Header.Get("Authorization"))
		}
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests <- req
		if req.OperationName != "GetTransactionsList" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"allTransactions":{"results":[{"id":"tx-1","date":"2026-05-08","amount":-20,"merchant":{"name":"Store"},"category":{"name":"Food"},"account":{"id":"acc-1"}}],"totalCount":1}}}`)
	}))
	defer server.Close()

	home := t.TempDir()
	sessionDir := filepath.Join(home, ".monarchmoney-cli")
	if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		sessionDir = filepath.Join(home, "monarchmoney-cli")
	}
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "session.json"), []byte(`{"token":"test-token"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := buildBinary(t)
	env := []string{"MONARCH_API_ENDPOINT=" + server.URL}
	stdout, code := runWithHome(t, bin, home, env, "transactions", "list", "--json", "--limit", "10", "--offset", "5", "--from", "2026-05-01", "--to", "2026-05-31", "--category-id", "food", "--pending=false")
	requireZero(t, code, stdout)
	var result struct {
		OK   bool `json:"ok"`
		Data struct {
			Total        int `json:"total"`
			Transactions []struct {
				ID       string  `json:"id"`
				Merchant string  `json:"merchant"`
				Amount   float64 `json:"amount"`
			} `json:"transactions"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode result: %v: %s", err, stdout)
	}
	if !result.OK || result.Data.Total != 1 || len(result.Data.Transactions) != 1 || result.Data.Transactions[0].ID != "tx-1" || result.Data.Transactions[0].Merchant != "Store" || result.Data.Transactions[0].Amount != -20 {
		t.Fatalf("unexpected transactions: %s", stdout)
	}
	select {
	case req := <-requests:
		want := map[string]any{"limit": float64(10), "offset": float64(5), "filters": map[string]any{"search": "", "categories": []any{"food"}, "accounts": []any{}, "tags": []any{}, "startDate": "2026-05-01", "endDate": "2026-05-31", "isPending": false}}
		if req.OperationName != "GetTransactionsList" || !reflect.DeepEqual(req.Variables, want) {
			t.Fatalf("unexpected GraphQL request: %+v", req)
		}
	default:
		t.Fatal("no GraphQL request received")
	}

	stdout, code = runWithHome(t, bin, home, env, "transactions", "update", "tx-1", "--category", "food", "--dry-run", "--json")
	requireZero(t, code, stdout)
	var plan struct {
		OK   bool `json:"ok"`
		Data struct {
			Mutations []struct {
				Operation  string         `json:"operation"`
				ResourceID string         `json:"resource_id"`
				After      map[string]any `json:"after"`
			} `json:"planned_mutations"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &plan); err != nil {
		t.Fatalf("decode plan: %v: %s", err, stdout)
	}
	if !plan.OK || len(plan.Data.Mutations) != 1 || plan.Data.Mutations[0].Operation != "transactions.update" || plan.Data.Mutations[0].ResourceID != "tx-1" || plan.Data.Mutations[0].After["categoryId"] != "food" {
		t.Fatalf("unexpected plan: %s", stdout)
	}
	stdout, code = runWithHome(t, bin, home, env, "transactions", "update", "tx-1", "--category", "food", "--json")
	var failure struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &failure); err != nil || code != 10 || failure.Error.Code != "CONFIRMATION_REQUIRED" {
		t.Fatalf("expected confirmation error, got exit %d: %s", code, stdout)
	}
	select {
	case req := <-requests:
		t.Fatalf("unexpected request from protected mutation: %+v", req)
	default:
	}
}
