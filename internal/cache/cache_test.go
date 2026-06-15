package cache

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStorePersistsAndQueriesData(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(filepath.Join(dir, "cache", "monarch.sqlite"))
	mustNoError(t, err, "NewStore()")
	defer store.Close() //nolint:errcheck // test cleanup

	accounts := []Account{{
		ID:             "acc_1",
		DisplayName:    "Checking",
		AccountType:    "cash",
		DisplayBalance: 1250.50,
		UpdatedAt:      time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
	}}
	mustNoError(t, store.SaveAccounts(accounts), "SaveAccounts()")

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
	mustNoError(t, store.SaveTransactions(transactions), "SaveTransactions()")

	morning := Transaction{
		ID:       "tx_3",
		Date:     time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
		Amount:   -5,
		Merchant: "Rent Cafe",
		Category: "Dining",
		Notes:    "morning rent chat",
	}
	mustNoError(t, store.SaveTransactions([]Transaction{morning}), "SaveTransactions(morning)")

	matches, err := store.SearchTransactions("Rent")
	mustNoError(t, err, "SearchTransactions()")
	assertSearchOrder(t, matches, "tx_3", "tx_2")

	stats, err := store.GetStats()
	mustNoError(t, err, "GetStats()")
	assertStat(t, stats, "accounts", 1)
	assertStat(t, stats, "transactions", 3)

	deleted, err := store.Cleanup("2026-05-02")
	mustNoError(t, err, "Cleanup()")
	if deleted != 1 {
		t.Fatalf("Cleanup() deleted = %d, want %d", deleted, 1)
	}

	stats, err = store.GetStats()
	mustNoError(t, err, "GetStats() after cleanup")
	assertStat(t, stats, "transactions", 2)
}

func TestSyncMetaTracksLastSync(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(filepath.Join(dir, "cache", "monarch.sqlite"))
	mustNoError(t, err, "NewStore()")
	defer store.Close() //nolint:errcheck // test cleanup

	// No sync yet.
	ls, err := store.LastSync()
	mustNoError(t, err, "LastSync()")
	if ls != nil {
		t.Fatalf("LastSync() = %v, want nil before first sync", ls)
	}

	// Record a sync.
	mustNoError(t, store.RecordSync(5, 100), "RecordSync()")

	ls, err = store.LastSync()
	mustNoError(t, err, "LastSync()")
	if ls == nil {
		t.Fatal("LastSync() = nil, want sync metadata")
	}
	if ls.Accounts != 5 || ls.TxCount != 100 {
		t.Fatalf("LastSync() = %+v, want accounts=5, transactions=100", ls)
	}

	// Stats should include last_synced_at.
	stats, err := store.GetStats()
	mustNoError(t, err, "GetStats()")
	if _, ok := stats["last_synced_at"]; !ok {
		t.Fatalf("GetStats() = %v, want last_synced_at key", stats)
	}
}

func mustNoError(t *testing.T, err error, call string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s error = %v", call, err)
	}
}

func assertSearchOrder(t *testing.T, matches []Transaction, firstID, secondID string) {
	t.Helper()
	if len(matches) != 2 {
		t.Fatalf("SearchTransactions() len = %d, want 2", len(matches))
	}
	if matches[0].ID != firstID || matches[1].ID != secondID {
		t.Fatalf("SearchTransactions() = %#v, want date-desc %s then %s", matches, firstID, secondID)
	}
}

func assertStat(t *testing.T, stats map[string]any, key string, want int64) {
	t.Helper()
	got, ok := stats[key]
	if !ok {
		t.Fatalf("%s missing from stats", key)
	}
	if got.(int64) != want {
		t.Fatalf("%s = %v, want %d", key, got, want)
	}
}
