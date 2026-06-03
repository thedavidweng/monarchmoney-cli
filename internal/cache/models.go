package cache

import (
	"time"

	"gorm.io/gorm"
)

type Account struct {
	ID             string `gorm:"primaryKey"`
	DisplayName    string
	AccountType    string
	DisplayBalance float64
	UpdatedAt      time.Time
}

type Transaction struct {
	ID        string    `gorm:"primaryKey"`
	Date      time.Time `gorm:"index"`
	Amount    float64
	Merchant  string `gorm:"index"`
	Category  string `gorm:"index"`
	Notes     string
	AccountID string `gorm:"index"`
}

// SyncMeta tracks the last successful sync timestamp.
type SyncMeta struct {
	ID       uint      `gorm:"primaryKey"`
	SyncedAt time.Time `json:"synced_at"`
	Accounts int       `json:"accounts"`
	TxCount  int       `json:"transactions"`
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Account{}, &Transaction{}, &SyncMeta{})
}
