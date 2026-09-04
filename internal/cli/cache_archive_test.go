package cli

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"

	"github.com/thedavidweng/monarchmoney-cli/internal/cache"
	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func TestCacheSyncStoresArchiveFidelity(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	cachePath := filepath.Join(dir, "cache.sqlite")
	exitCode := withCacheCommandTestDefaults(t, sessionPath, cachePath)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}

		switch gqlReq.OperationName {
		case "GetAccounts":
			return testutil.JSONResponse(`{"data":{"accounts":[{"id":"acc_1","displayName":"Brokerage","type":{"name":"brokerage","display":"Brokerage","group":"investment"},"subtype":{"name":"default","display":"Default"},"displayBalance":9500.5,"currentBalance":9400.25,"updatedAt":"2026-05-09T10:00:00Z","deactivatedAt":"2026-05-09T10:00:00Z","isManual":true,"isHidden":true}]}}`), nil
		case "GetTransactionsList":
			return testutil.JSONResponse(`{"data":{"allTransactions":{"results":[{"id":"tx_1","date":"2026-05-09","amount":-12.34,"pending":true,"hideFromReports":true,"isRecurring":true,"dataProviderDescription":"Blue Bottle Coffee","plaidName":"BLUE BOTTLE COFFEE LLC","notes":"latte","reviewStatus":"flagged","needsReview":true,"isSplitTransaction":true,"splitTransactions":[{"id":"sp_1","amount":-10,"notes":"latte","category":{"name":"Dining"},"merchant":{"name":"Coffee"}},{"id":"sp_2","amount":-2.34,"notes":"tip","category":{"name":"Dining"},"merchant":{"name":"Coffee"}}],"category":{"id":"cat_1","name":"Dining","group":{"id":"g1","name":"Food & Drink","type":"expense"}},"merchant":{"name":"Coffee","id":"m1"},"account":{"id":"acc_1","displayName":"Checking","order":0,"type":{"group":"asset"}},"goal":{"id":"goal_1","name":"House"},"tags":[{"id":"tag-1","name":"work"}]}]}}}`), nil
		case "Web_GetHoldings":
			return testutil.JSONResponse(`{"data":{"portfolio":{"aggregateHoldings":{"edges":[{"node":{"id":"n1","quantity":10,"basis":900,"totalValue":1000,"holdings":[{"id":"h_1","quantity":10.5,"name":"Vanguard Total Stock Market ETF","ticker":"VTI","value":3000.5,"costBasis":2500,"account":{"id":"acc_1"}}]}}]}}}}`), nil
		default:
			t.Fatalf("unexpected operation %q", gqlReq.OperationName)
		}
		return nil, nil
	})

	out := captureStdout(t, func() {
		cacheSyncCmd.Run(cacheSyncCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}

	db, err := sql.Open("sqlite3", cachePath)
	if err != nil {
		t.Fatalf("open synced cache: %v", err)
	}
	defer db.Close()

	var typeGroup string
	var currentBalance float64
	var isManual, isHidden, isClosed int
	if err := db.QueryRow(
		`SELECT type_group, current_balance, is_manual, is_hidden, is_closed FROM accounts WHERE id = 'acc_1'`,
	).Scan(&typeGroup, &currentBalance, &isManual, &isHidden, &isClosed); err != nil {
		t.Fatalf("query cached account: %v", err)
	}
	if typeGroup != "investment" || currentBalance != 9400.25 || isManual != 1 || isHidden != 1 || isClosed != 1 {
		t.Fatalf("cached account archive fields incomplete: %q %v %d/%d/%d", typeGroup, currentBalance, isManual, isHidden, isClosed)
	}

	var plaidName, providerDescription, categoryGroup, groupType, reviewStatus string
	var pending, needsReview, hideFromReports, isRecurring int
	var goalID, goalName string
	if err := db.QueryRow(
		`SELECT plaid_name, provider_description, category_group, category_group_type, pending, hide_from_reports, is_recurring, review_status, needs_review, goal_id, goal_name FROM transactions WHERE id = 'tx_1'`,
	).Scan(&plaidName, &providerDescription, &categoryGroup, &groupType, &pending, &hideFromReports, &isRecurring, &reviewStatus, &needsReview, &goalID, &goalName); err != nil {
		t.Fatalf("query cached transaction: %v", err)
	}
	if plaidName != "BLUE BOTTLE COFFEE LLC" || providerDescription != "Blue Bottle Coffee" {
		t.Fatalf("raw merchant names = %q / %q", plaidName, providerDescription)
	}
	if categoryGroup != "Food & Drink" || groupType != "expense" {
		t.Fatalf("category group = %q (%q)", categoryGroup, groupType)
	}
	if pending != 1 || needsReview != 1 || hideFromReports != 1 || isRecurring != 1 || reviewStatus != "flagged" || goalID != "goal_1" || goalName != "House" {
		t.Fatalf("transaction review/goal state incomplete: %d %q %d %d %d %q %q", pending, reviewStatus, needsReview, hideFromReports, isRecurring, goalID, goalName)
	}

	assertCachedRowCount(t, db, `SELECT COUNT(*) FROM transaction_tags WHERE transaction_id = 'tx_1' AND name = 'work'`, 1, "tag rows")
	assertCachedRowCount(t, db, `SELECT COUNT(*) FROM transaction_splits WHERE transaction_id = 'tx_1'`, 2, "split rows")
	assertCachedRowCount(t, db, `SELECT COUNT(*) FROM transaction_splits WHERE id = 'sp_2' AND amount = -2.34 AND notes = 'tip'`, 1, "split detail")

	var ticker string
	var quantity, basis, value float64
	var holdingAccount string
	if err := db.QueryRow(
		`SELECT ticker, quantity, basis, value, account_id FROM holdings WHERE id = 'h_1'`,
	).Scan(&ticker, &quantity, &basis, &value, &holdingAccount); err != nil {
		t.Fatalf("query cached holding: %v", err)
	}
	if ticker != "VTI" || quantity != 10.5 || basis != 2500 || value != 3000.5 || holdingAccount != "acc_1" {
		t.Fatalf("cached holding incomplete: %q %v %v %v %q", ticker, quantity, basis, value, holdingAccount)
	}

	store, err := cache.NewStore(cachePath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}
	assertStatValue(t, stats, "holdings", 1)
}

func assertCachedRowCount(t *testing.T, db *sql.DB, query string, want int64, label string) {
	t.Helper()
	var got int64
	if err := db.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("query %s: %v", label, err)
	}
	if got != want {
		t.Fatalf("%s = %d, want %d (%s)", label, got, want, query)
	}
}

func assertStatValue(t *testing.T, stats map[string]any, key string, want int64) {
	t.Helper()
	got, ok := stats[key].(int64)
	if !ok || got != want {
		t.Fatalf("stats[%s] = %v, want %d", key, stats[key], want)
	}
}

func TestCacheSyncRebuildsOutdatedCache(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	cachePath := filepath.Join(dir, "cache.sqlite")
	exitCode := withCacheCommandTestDefaults(t, sessionPath, cachePath)
	saveTestSession(t, sessionPath)
	createLegacyCacheFile(t, cachePath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}

		switch gqlReq.OperationName {
		case "GetAccounts":
			return testutil.JSONResponse(`{"data":{"accounts":[{"id":"acc_new","displayName":"Checking","type":{"name":"cash","group":"asset"},"subtype":{"name":"checking"},"displayBalance":10,"currentBalance":10,"updatedAt":"2026-05-09T10:00:00Z"}]}}`), nil
		case "GetTransactionsList":
			return testutil.JSONResponse(`{"data":{"allTransactions":{"results":[{"id":"tx_new","date":"2026-05-09","amount":-5,"merchant":{"name":"Cafe"},"category":{"name":"Dining"},"account":{"id":"acc_new"}}],"totalCount":1}}}`), nil
		case "Web_GetHoldings":
			return testutil.JSONResponse(`{"data":{"portfolio":{"aggregateHoldings":{"edges":[]}}}}`), nil
		default:
			t.Fatalf("unexpected operation %q", gqlReq.OperationName)
		}
		return nil, nil
	})

	out := captureStdout(t, func() {
		cacheSyncCmd.Run(cacheSyncCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}

	store, err := cache.NewStore(cachePath)
	if err != nil {
		t.Fatalf("NewStore() after rebuild error = %v", err)
	}
	defer store.Close()

	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}
	assertStatValue(t, stats, "accounts", 1)
	if _, ok := stats["last_synced_at"]; !ok {
		t.Fatalf("stats = %v, want last_synced_at recorded by sync", stats)
	}
}

func TestCacheSearchOnOutdatedCachePromptsResync(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	cachePath := filepath.Join(dir, "cache.sqlite")
	exitCode := withCacheCommandTestDefaults(t, sessionPath, cachePath)
	createLegacyCacheFile(t, cachePath)

	out := captureStdout(t, func() {
		cacheSearchCmd.Run(cacheSearchCmd, []string{"coffee"})
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want failure on outdated cache; output=%q", out)
	}
	if !strings.Contains(out, "run 'monarch cache sync' to rebuild it") {
		t.Fatalf("output = %q, want re-sync guidance", out)
	}
}

func createLegacyCacheFile(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open legacy cache: %v", err)
	}
	defer db.Close()
	const legacy = `
CREATE TABLE IF NOT EXISTS accounts (
	id TEXT PRIMARY KEY, display_name TEXT, account_type TEXT, display_balance REAL, updated_at TEXT
);
CREATE TABLE IF NOT EXISTS transactions (
	id TEXT PRIMARY KEY, date TEXT, amount REAL, merchant TEXT, category TEXT, notes TEXT, account_id TEXT
);
CREATE TABLE IF NOT EXISTS sync_meta (
	id INTEGER PRIMARY KEY AUTOINCREMENT, synced_at TEXT, accounts INTEGER, tx_count INTEGER
);
INSERT INTO accounts (id, display_name) VALUES ('acc_old', 'Legacy');
`
	if _, err := db.Exec(legacy); err != nil {
		t.Fatalf("create legacy cache schema: %v", err)
	}
}
