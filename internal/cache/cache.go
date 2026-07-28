package cache

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
)

var errEmptySlice = errors.New("empty slice found")

type Store struct {
	db *sql.DB
}

var mkdirAll = os.MkdirAll
var openDB = func(path string) (*sql.DB, error) {
	return sql.Open("sqlite3", path)
}
var migrateStore = Migrate

func NewStore(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := mkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	db, err := openDB(path)
	if err != nil {
		return nil, err
	}

	// Enable WAL so a read (e.g. cache search) is safe while a cache sync writes.
	// This first statement also forces the database file to exist, so its
	// permissions can be locked down before any data is written.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := os.Chmod(path, 0o600); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := migrateStore(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) SaveAccounts(accounts []Account) error {
	if len(accounts) == 0 {
		return errEmptySlice
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	const stmt = `INSERT OR REPLACE INTO accounts (id, display_name, account_type, display_balance, updated_at) VALUES (?, ?, ?, ?, ?)`
	for i := range accounts {
		a := &accounts[i]
		if _, err := tx.Exec(stmt, a.ID, a.DisplayName, a.AccountType, a.DisplayBalance, a.UpdatedAt.UTC().Format(time.RFC3339)); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) SaveTransactions(txs []Transaction) error {
	if len(txs) == 0 {
		return errEmptySlice
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	const stmt = `INSERT OR REPLACE INTO transactions (id, date, amount, merchant, category, notes, account_id) VALUES (?, ?, ?, ?, ?, ?, ?)`
	for i := range txs {
		t := &txs[i]
		if _, err := tx.Exec(stmt, t.ID, t.Date.UTC().Format(time.RFC3339), t.Amount, t.Merchant, t.Category, t.Notes, t.AccountID); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) RecordSync(accountCount, txCount int) error {
	_, err := s.db.Exec(
		`INSERT INTO sync_meta (synced_at, accounts, tx_count) VALUES (?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339), accountCount, txCount,
	)
	return err
}

func (s *Store) LastSync() (*SyncMeta, error) {
	row := s.db.QueryRow(`SELECT id, synced_at, accounts, tx_count FROM sync_meta ORDER BY synced_at DESC, id DESC LIMIT 1`)
	var (
		id       uint
		syncedAt string
		accounts int
		txCount  int
	)
	if err := row.Scan(&id, &syncedAt, &accounts, &txCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	parsed, err := time.Parse(time.RFC3339, syncedAt)
	if err != nil {
		return nil, err
	}
	return &SyncMeta{ID: id, SyncedAt: parsed, Accounts: accounts, TxCount: txCount}, nil
}

func (s *Store) SearchTransactions(query string) ([]Transaction, error) {
	like := "%" + query + "%"
	rows, err := s.db.Query(
		`SELECT id, date, amount, merchant, category, notes, account_id
		 FROM transactions
		 WHERE merchant LIKE ? OR notes LIKE ? OR category LIKE ?
		 ORDER BY date DESC, id ASC`,
		like, like, like,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var txs []Transaction
	for rows.Next() {
		var (
			t    Transaction
			date string
		)
		if err := rows.Scan(&t.ID, &date, &t.Amount, &t.Merchant, &t.Category, &t.Notes, &t.AccountID); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339, date)
		if err != nil {
			return nil, err
		}
		t.Date = parsed
		txs = append(txs, t)
	}
	return txs, rows.Err()
}

func (s *Store) Cleanup(before string) (int64, error) {
	result, err := s.db.Exec(`DELETE FROM transactions WHERE date < ?`, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) GetStats() (map[string]any, error) {
	var accCount, txCount int64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&accCount); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&txCount); err != nil {
		return nil, err
	}

	stats := map[string]any{
		"accounts":     accCount,
		"transactions": txCount,
	}

	lastSync, err := s.LastSync()
	if err != nil {
		return nil, err
	}
	if lastSync != nil {
		stats["last_synced_at"] = lastSync.SyncedAt.Format(time.RFC3339)
	}

	return stats, nil
}
