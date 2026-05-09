package cache

import (
	"os"
	"path/filepath"

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

func (s *Store) GetStats() (map[string]int64, error) {
	var accCount, txCount int64
	s.db.Model(&Account{}).Count(&accCount)
	s.db.Model(&Transaction{}).Count(&txCount)
	return map[string]int64{
		"accounts":     accCount,
		"transactions": txCount,
	}, nil
}
