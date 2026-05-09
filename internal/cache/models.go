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

type SyncState struct {
	Key      string `gorm:"primaryKey"`
	LastSync time.Time
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Account{}, &Transaction{}, &SyncState{})
}
