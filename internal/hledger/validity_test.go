package hledger

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/cache"
)

func TestGenerateValidHledgerJournal(t *testing.T) {
	hledger, err := exec.LookPath("hledger")
	if err != nil {
		t.Skip("hledger not installed")
	}
	data := Data{
		Accounts: []cache.Account{
			{ID: "acc_1", DisplayName: "Checking", TypeGroup: "asset", CurrentBalance: 379.25},
			{ID: "acc_2", DisplayName: "Credit Card", TypeGroup: "liability", CurrentBalance: -100},
			{ID: "acc_3", DisplayName: "Brokerage", TypeGroup: "investment"},
			{ID: "acc_4", DisplayName: "Savings", TypeGroup: "asset", IsClosed: true, CurrentBalance: 500},
		},
		Transactions: []cache.Transaction{
			{ID: "tx_1", Date: d("2026-05-01"), Amount: 2500, Merchant: "Employer Inc", Category: "Paycheck", CategoryGroupType: "income", AccountID: "acc_1"},
			{ID: "tx_0", Date: d("2026-05-01"), Amount: -440.5, Merchant: "Restaurant", Category: "Dining", CategoryGroupType: "expense", AccountID: "acc_2"},
			{
				ID: "tx_2", Date: d("2026-05-02"), Amount: -80.25, Merchant: "Grocery Store", Category: "Groceries", CategoryGroupType: "expense", AccountID: "acc_1",
				Splits: []cache.Split{{ID: "sp_1", Amount: -50.25, Category: "Groceries"}, {ID: "sp_2", Amount: -30, Category: "Household"}},
			},
			{ID: "tx_3", Date: d("2026-05-03"), Amount: -500, Merchant: "To Savings", Category: "Transfer", CategoryGroupType: "transfer", AccountID: "acc_1"},
			{ID: "tx_4", Date: d("2026-05-03"), Amount: 500, Merchant: "From Checking", Category: "Transfer", CategoryGroupType: "transfer", AccountID: "acc_4"},
			{ID: "tx_5", Date: d("2026-05-04"), Amount: -340.5, Merchant: "Card Payment", Category: "Credit Card Payment", CategoryGroupType: "transfer", AccountID: "acc_1"},
			{ID: "tx_6", Date: d("2026-05-04"), Amount: 340.5, Merchant: "Payment Received", Category: "Credit Card Payment", CategoryGroupType: "transfer", AccountID: "acc_2"},
			{ID: "tx_7", Date: d("2026-05-05"), Amount: -1200, Merchant: "To Brokerage", Category: "Transfer", CategoryGroupType: "transfer", AccountID: "acc_1"},
			{ID: "tx_p", Date: d("2026-05-06"), Amount: -99, Merchant: "Pending Charge", Category: "Dining", CategoryGroupType: "expense", Pending: true, AccountID: "acc_1"},
		},
		Holdings: []cache.Holding{
			{ID: "h_1", Ticker: "VTI", Quantity: 3.5, Basis: 1000, Value: 1102.5, AccountID: "acc_3"},
			{ID: "h_2", Ticker: "CUR:USD", Quantity: 97.5, Basis: 97.5, Value: 97.5, AccountID: "acc_3"},
		},
		Anchor: d("2026-05-07"),
	}
	journal := Generate(&data)
	path := filepath.Join(t.TempDir(), "monarch.journal")
	if err := os.WriteFile(path, []byte(journal), 0o600); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) string {
		cmd := exec.Command(hledger, append([]string{"-f", path}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("hledger %v failed: %v\n%s", args, err, out)
		}
		return string(out)
	}

	bal := run("bal")
	if !strings.Contains(bal, "379.25") {
		t.Fatalf("checking balance missing from:\n%s", bal)
	}
	run("reg")
	run("bal", "-V")
	run("check")
}
