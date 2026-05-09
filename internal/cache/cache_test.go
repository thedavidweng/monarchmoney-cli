package cache

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStorePersistsAndQueriesData(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(filepath.Join(dir, "cache", "monarch.sqlite"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	accounts := []Account{{
		ID:             "acc_1",
		DisplayName:    "Checking",
		AccountType:    "cash",
		DisplayBalance: 1250.50,
		UpdatedAt:      time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
	}}
	if err := store.SaveAccounts(accounts); err != nil {
		t.Fatalf("SaveAccounts() error = %v", err)
	}

	transactions := []Transaction{
		{
			ID:        "tx_1",
			Date:      time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			Amount:    -42.75,
			Merchant:  "Coffee Shop",
			Category:  "Dining",
			Notes:     "Morning latte",
			AccountID: "acc_1",
		},
		{
			ID:        "tx_2",
			Date:      time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
			Amount:    -1200,
			Merchant:  "Rent",
			Category:  "Housing",
			Notes:     "May rent",
			AccountID: "acc_1",
		},
	}
	if err := store.SaveTransactions(transactions); err != nil {
		t.Fatalf("SaveTransactions() error = %v", err)
	}

	morning := Transaction{
		ID:       "tx_3",
		Date:     time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
		Amount:   -5,
		Merchant: "Rent Cafe",
		Category: "Dining",
		Notes:    "morning rent chat",
	}
	if err := store.SaveTransactions([]Transaction{morning}); err != nil {
		t.Fatalf("SaveTransactions(morning) error = %v", err)
	}

	matches, err := store.SearchTransactions("Rent")
	if err != nil {
		t.Fatalf("SearchTransactions() error = %v", err)
	}
	if len(matches) != 2 || matches[0].ID != "tx_3" || matches[1].ID != "tx_2" {
		t.Fatalf("SearchTransactions() = %#v, want date-desc tx_3 then tx_2", matches)
	}

	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}
	if got, want := stats["accounts"], int64(1); got != want {
		t.Fatalf("accounts = %d, want %d", got, want)
	}
	if got, want := stats["transactions"], int64(3); got != want {
		t.Fatalf("transactions = %d, want %d", got, want)
	}

	deleted, err := store.Cleanup("2026-05-02")
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if got, want := deleted, int64(1); got != want {
		t.Fatalf("Cleanup() deleted = %d, want %d", got, want)
	}

	stats, err = store.GetStats()
	if err != nil {
		t.Fatalf("GetStats() after cleanup error = %v", err)
	}
	if got, want := stats["transactions"], int64(2); got != want {
		t.Fatalf("transactions after cleanup = %d, want %d", got, want)
	}
}
