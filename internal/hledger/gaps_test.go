package hledger

import (
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/cache"
)

func TestBalanceGaps(t *testing.T) {
	data := &Data{
		Accounts: []cache.Account{
			{ID: "acc_1", DisplayName: "Checking", TypeGroup: "asset", CurrentBalance: 100},
			{ID: "acc_2", DisplayName: "Reconciled", TypeGroup: "asset", CurrentBalance: -50},
			{ID: "acc_3", DisplayName: "Brokerage", TypeGroup: "investment", CurrentBalance: 9999},
		},
		Transactions: []cache.Transaction{
			{ID: "tx_1", Date: d("2026-05-01"), Amount: -50, Merchant: "Store", Category: "Food", CategoryGroupType: "expense", AccountID: "acc_2"},
		},
	}
	gaps := BalanceGaps(data)
	if len(gaps) != 1 || gaps[0].Account != "assets:monarch:checking" || gaps[0].Cents != 10000 {
		t.Fatalf("BalanceGaps() = %+v", gaps)
	}
}
