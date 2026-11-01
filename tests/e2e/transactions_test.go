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

	home := authenticatedHome(t)
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

func authenticatedHome(t *testing.T) string {
	t.Helper()
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
	return home
}

func TestAnalyzeAnomaliesAcrossTransactionPages(t *testing.T) {
	requests := make(chan map[string]any, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Token test-token" {
			t.Errorf("unexpected request: method=%s authorization=%q", r.Method, r.Header.Get("Authorization"))
		}
		var req struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OperationName != "GetTransactionsList" {
			t.Errorf("unexpected GraphQL request: %+v, error: %v", req, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		select {
		case requests <- req.Variables:
		default:
			t.Error("unexpected extra GraphQL page request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch req.Variables["offset"] {
		case float64(0):
			fmt.Fprint(w, `{"data":{"allTransactions":{"results":[{"id":"h1","date":"2026-03-10","amount":-100,"category":{"name":"Dining"}},{"id":"h2","date":"2026-04-10","amount":-100,"category":{"name":"Dining"}}],"totalCount":4}}}`)
		case float64(2):
			fmt.Fprint(w, `{"data":{"allTransactions":{"results":[{"id":"c1","date":"2026-05-10","amount":-300,"category":{"name":"Dining"},"merchant":{"name":"Restaurant"}},{"id":"income","date":"2026-05-11","amount":1000,"category":{"name":"Dining"}}],"totalCount":4}}}`)
		default:
			t.Errorf("unexpected offset: %v", req.Variables["offset"])
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	stdout, code := runWithHome(t, buildBinary(t), authenticatedHome(t), []string{"MONARCH_API_ENDPOINT=" + server.URL}, "analyze", "anomalies", "--month", "2026-05", "--history-months", "2", "--json")
	requireZero(t, code, stdout)
	var result struct {
		OK   bool `json:"ok"`
		Data struct {
			Period    map[string]string `json:"period"`
			Anomalies []struct {
				Category        string  `json:"category"`
				CurrentMonth    float64 `json:"current_month"`
				AvgHistory      float64 `json:"avg_history"`
				Ratio           float64 `json:"ratio"`
				Severity        string  `json:"severity"`
				LargestMerchant string  `json:"largest_merchant"`
				LargestAmount   float64 `json:"largest_amount"`
			} `json:"anomalies"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode analysis: %v: %s", err, stdout)
	}
	if !result.OK || !reflect.DeepEqual(result.Data.Period, map[string]string{"start_date": "2026-05-01", "end_date": "2026-05-31"}) || len(result.Data.Anomalies) != 1 {
		t.Fatalf("unexpected analysis: %s", stdout)
	}
	a := result.Data.Anomalies[0]
	if a.Category != "Dining" || a.CurrentMonth != 300 || a.AvgHistory != 100 || a.Ratio != 3 || a.Severity != "high" || a.LargestMerchant != "Restaurant" || a.LargestAmount != 300 {
		t.Fatalf("unexpected anomaly: %+v", a)
	}
	for _, offset := range []float64{0, 2} {
		select {
		case variables := <-requests:
			filters, ok := variables["filters"].(map[string]any)
			if !ok || variables["offset"] != offset || variables["limit"] != float64(1000) || filters["startDate"] != "2026-03-01" || filters["endDate"] != "2026-05-31" {
				t.Fatalf("unexpected page request: %+v", variables)
			}
		default:
			t.Fatal("missing GraphQL page request")
		}
	}
}
