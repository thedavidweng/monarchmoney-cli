package cache

import (
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

var mkdirAll = os.MkdirAll
var openSQLite = func(path string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(path), &gorm.Config{})
}
var migrateStore = Migrate

func NewStore(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := mkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	db, err := openSQLite(path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0600); err != nil {
		return nil, err
	}

	// Enable WAL mode for safe concurrent reads.
	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		return nil, err
	}

	if err := migrateStore(db); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) SaveAccounts(accounts []Account) error {
	return s.db.Save(&accounts).Error
}

func (s *Store) SaveTransactions(txs []Transaction) error {
	return s.db.Save(&txs).Error
}

// RecordSync records the timestamp and counts of a successful sync.
func (s *Store) RecordSync(accountCount, txCount int) error {
	return s.db.Create(&SyncMeta{
		SyncedAt: time.Now().UTC(),
		Accounts: accountCount,
		TxCount:  txCount,
	}).Error
}

// LastSync returns the most recent sync metadata, or nil if never synced.
func (s *Store) LastSync() (*SyncMeta, error) {
	var meta SyncMeta
	if err := s.db.Order("synced_at DESC").First(&meta).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &meta, nil
}

func (s *Store) SearchTransactions(query string) ([]Transaction, error) {
	var txs []Transaction
	err := s.db.Where("merchant LIKE ? OR notes LIKE ? OR category LIKE ?", "%"+query+"%", "%"+query+"%", "%"+query+"%").
		Order("date DESC").
		Order("id ASC").
		Find(&txs).Error
	return txs, err
}

func (s *Store) Cleanup(before string) (int64, error) {
	result := s.db.Where("date < ?", before).Delete(&Transaction{})
	return result.RowsAffected, result.Error
}

func (s *Store) GetStats() (map[string]interface{}, error) {
	var accCount, txCount int64
	s.db.Model(&Account{}).Count(&accCount)
	s.db.Model(&Transaction{}).Count(&txCount)

	stats := map[string]interface{}{
		"accounts":     accCount,
		"transactions": txCount,
	}

	lastSync, _ := s.LastSync()
	if lastSync != nil {
		stats["last_synced_at"] = lastSync.SyncedAt.Format(time.RFC3339)
	}

	return stats, nil
}
