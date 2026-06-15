package cache

import (
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkSearchTransactions(b *testing.B) {
	dir := b.TempDir()
	store, err := NewStore(filepath.Join(dir, "bench.sqlite"))
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close() //nolint:errcheck // bench cleanup

	// Seed data
	txs := make([]Transaction, 1000)
	for i := range txs {
		txs[i] = Transaction{
			Date:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Merchant: "Merchant " + string(rune('A'+i%26)),
			Category: "Category",
			Amount:   float64(i),
			Notes:    "some notes for transaction",
		}
	}
	if err := store.SaveTransactions(txs); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := store.SearchTransactions("Merchant")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetStats(b *testing.B) {
	dir := b.TempDir()
	store, err := NewStore(filepath.Join(dir, "bench.sqlite"))
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close() //nolint:errcheck // bench cleanup

	txs := make([]Transaction, 100)
	for i := range txs {
		txs[i] = Transaction{Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Merchant: "M", Amount: 1.0}
	}
	store.SaveTransactions(txs) //nolint:errcheck // bench seed data

	b.ResetTimer()
	for b.Loop() {
		_, err := store.GetStats()
		if err != nil {
			b.Fatal(err)
		}
	}
}
