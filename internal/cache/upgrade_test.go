package cache

import (
	"database/sql"
	"testing"
	"time"
)

func TestMidVersionArchiveUpgradesInPlace(t *testing.T) {
	path := t.TempDir() + "/c.sqlite"
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE accounts (id TEXT PRIMARY KEY, display_name TEXT, account_type TEXT, type_group TEXT,
		display_balance REAL, current_balance REAL, is_manual INTEGER, is_hidden INTEGER, is_closed INTEGER, updated_at TEXT);
		CREATE TABLE transactions (id TEXT PRIMARY KEY, date TEXT, amount REAL, merchant TEXT, plaid_name TEXT,
		provider_description TEXT, category TEXT, category_group TEXT, category_group_type TEXT, notes TEXT,
		pending INTEGER, review_status TEXT, needs_review INTEGER, goal_id TEXT, goal_name TEXT, account_id TEXT);
		CREATE TABLE transaction_tags (transaction_id TEXT, tag_id TEXT, name TEXT, PRIMARY KEY (transaction_id, tag_id));
		CREATE TABLE transaction_splits (id TEXT PRIMARY KEY, transaction_id TEXT, amount REAL, category TEXT, merchant TEXT, notes TEXT);
		CREATE TABLE holdings (id TEXT PRIMARY KEY, ticker TEXT, name TEXT, quantity REAL, basis REAL, value REAL, account_id TEXT);
		CREATE TABLE sync_meta (id INTEGER PRIMARY KEY AUTOINCREMENT, synced_at TEXT, accounts INTEGER, tx_count INTEGER);`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO transactions (id, date, amount, merchant, plaid_name, provider_description, category, category_group, category_group_type, notes, pending, review_status, needs_review, goal_id, goal_name, account_id) VALUES ('tx_1', '2026-05-09T00:00:00Z', -12.34, 'Coffee', '', '', '', '', '', '', 0, '', 0, '', '', 'acc_1')`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() on mid-version archive: %v", err)
	}
	defer store.Close()
	txs, err := store.Transactions()
	if err != nil {
		t.Fatalf("Transactions(): %v", err)
	}
	if len(txs) != 1 || txs[0].ID != "tx_1" || txs[0].HideFromReports || txs[0].IsRecurring {
		t.Fatalf("pre-existing rows not preserved/upgraded: %+v", txs)
	}
	parsed, _ := time.Parse(time.RFC3339, "2026-05-09T00:00:00Z")
	if err := store.SaveTransactions([]Transaction{{ID: "tx_1", Date: parsed, Amount: -12.34, Merchant: "Coffee", AccountID: "acc_1", HideFromReports: true}}); err != nil {
		t.Fatalf("SaveTransactions(): %v", err)
	}
	got, err := store.Transactions()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].HideFromReports || got[0].IsRecurring {
		t.Fatalf("upsert after upgrade wrong: %+v", got)
	}
}
