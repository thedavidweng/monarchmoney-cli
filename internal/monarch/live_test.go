package monarch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

const liveWritesEnv = "MONARCH_LIVE_WRITES"

func liveEnvBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func TestLiveEndpointAvailability(t *testing.T) {
	token := os.Getenv("MONARCH_LIVE_TOKEN")
	if token == "" {
		t.Skip("MONARCH_LIVE_TOKEN not set; skipping live endpoint availability checks")
	}
	endpoint := os.Getenv("MONARCH_LIVE_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.monarch.com/graphql"
	}

	svc := NewService(graphql.NewClient(endpoint, token, 60*time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	p := &liveProbe{t: t, svc: svc, ctx: ctx}
	p.identity()
	p.accounts()
	p.transactions()
	p.rules()
	p.budgets()
	p.cashflow()
	p.categories()
	p.goals()
	p.investments()
	p.recurring()
	p.tags()
	p.institutions()
	p.credit()
	p.subscription()
	p.household()
	p.reports()
	p.merchants()
	p.receipts()
	p.debt()
	p.syncHealth()
	p.whoami()

	if liveEnvBool(liveWritesEnv) {
		p.ruleRoundtrip()
		p.ruleReorderNoop()
		p.balanceHistoryUpload()
		p.attachmentUpload()
		p.receiptUpload()
	} else {
		t.Logf("%s not set; skipped write/upload endpoint probes", liveWritesEnv)
	}
}

type liveProbe struct {
	t   *testing.T
	svc *Service
	ctx context.Context
}

func (p *liveProbe) check(name string, call func() error) {
	p.t.Run(name, func(t *testing.T) {
		if err := call(); err != nil {
			t.Errorf("endpoint unavailable or schema changed: %v", err)
		}
	})
}

func (p *liveProbe) monthStart() string {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
}

func (p *liveProbe) today() string {
	return time.Now().Format("2006-01-02")
}

func (p *liveProbe) identity() {
	var resp struct {
		Me struct {
			Email string `json:"email"`
		} `json:"me"`
	}
	p.check("identity/GetIdentity", func() error {
		err := p.svc.Client.Do(p.ctx, &graphql.Request{
			OperationName: "GetIdentity",
			Query:         queries.Get("GetIdentity.graphql"),
		}, &resp)
		if err != nil {
			return err
		}
		if resp.Me.Email == "" {
			return fmt.Errorf("GetIdentity returned empty email")
		}
		return nil
	})
}

func (p *liveProbe) accounts() {
	start, end := p.monthStart(), p.today()

	var accounts []Account
	p.check("accounts/ListAccounts", func() error {
		var err error
		accounts, err = p.svc.ListAccounts(p.ctx)
		return err
	})

	accountID := ""
	if len(accounts) > 0 {
		accountID = accounts[0].ID
	} else {
		p.t.Log("no accounts; skipping account-scoped probes")
	}

	p.check("accounts/GetAccountTypes", func() error { _, err := p.svc.GetAccountTypes(p.ctx); return err })
	p.check("accounts/GetAccountRecentBalances", func() error { _, err := p.svc.GetAccountRecentBalances(p.ctx, start); return err })
	p.check("accounts/GetAccountBalancesAt", func() error { _, err := p.svc.GetAccountBalancesAt(p.ctx, end, nil); return err })
	p.check("accounts/GetSnapshotsByAccountType", func() error { _, err := p.svc.GetSnapshotsByAccountType(p.ctx, start, "month"); return err })
	p.check("accounts/GetAggregateSnapshots", func() error { _, err := p.svc.GetAggregateSnapshots(p.ctx, start, end, ""); return err })
	p.check("accounts/GetAccountsRefreshStatus", func() error { _, err := p.svc.GetAccountsRefreshStatus(p.ctx); return err })

	if accountID == "" {
		return
	}
	p.check("accounts/GetAccount", func() error { _, err := p.svc.GetAccount(p.ctx, accountID); return err })
	p.check("accounts/GetAccountHoldings", func() error { _, err := p.svc.GetAccountHoldings(p.ctx, accountID); return err })
	p.check("accounts/GetAccountHistory", func() error { _, err := p.svc.GetAccountHistory(p.ctx, accountID, start, end); return err })
}

func (p *liveProbe) transactions() {
	start, end := p.monthStart(), p.today()

	var txs []Transaction
	p.check("transactions/ListTransactions", func() error {
		var err error
		txs, _, err = p.svc.ListTransactions(p.ctx, &ListTransactionsOptions{Limit: 5, StartDate: start, EndDate: end})
		return err
	})

	txID := ""
	if len(txs) > 0 {
		txID = txs[0].ID
	} else {
		p.t.Log("no transactions; skipping transaction-scoped probes")
	}

	p.check("transactions/GetTransactionsSummary", func() error { _, err := p.svc.GetTransactionsSummary(p.ctx, start, end); return err })
	p.check("transactions/ListTransactionAttachments", func() error {
		if txID == "" {
			return nil
		}
		attachments, err := p.svc.ListTransactionAttachments(p.ctx, txID)
		if err != nil {
			return err
		}
		if len(attachments) > 0 {
			if _, err := p.svc.GetTransactionAttachment(p.ctx, attachments[0].ID); err != nil {
				return err
			}
		}
		return nil
	})

	if txID == "" {
		return
	}
	p.check("transactions/GetTransaction", func() error { _, err := p.svc.GetTransaction(p.ctx, txID); return err })
	p.check("transactions/GetTransactionSplits", func() error { _, err := p.svc.GetTransactionSplits(p.ctx, txID); return err })
}

func (p *liveProbe) duplicates() {
	end := p.today()
	start := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	p.check("transactions/GetDuplicateTransactions", func() error { _, err := p.svc.GetDuplicateTransactions(p.ctx, start, end); return err })
}

func (p *liveProbe) rules() {
	p.check("rules/ListRules", func() error { _, err := p.svc.ListRules(p.ctx); return err })
}

func (p *liveProbe) budgets() {
	start, end := p.monthStart(), p.today()
	p.check("budgets/ListBudgets", func() error {
		_, err := p.svc.ListBudgets(p.ctx, ListBudgetsOptions{StartDate: start, EndDate: end})
		return err
	})
	p.check("budgets/GetBudgetSettings", func() error { _, err := p.svc.GetBudgetSettings(p.ctx); return err })
	p.check("budgets/GetFlexRolloverSettings", func() error { _, err := p.svc.GetFlexRolloverSettings(p.ctx); return err })

	categories, err := p.svc.ListCategories(p.ctx)
	if err != nil || len(categories) == 0 {
		p.t.Log("categories unavailable; skipping budget show probe")
		return
	}
	categoryID := categories[0].ID
	p.check("budgets/GetBudget", func() error { _, err := p.svc.GetBudget(p.ctx, categoryID, start, end); return err })
}

func (p *liveProbe) cashflow() {
	start, end := p.monthStart(), p.today()
	p.check("cashflow/ListCashflow", func() error { _, err := p.svc.ListCashflow(p.ctx, start, end); return err })
	p.check("cashflow/GetCashflowSummary", func() error { _, err := p.svc.GetCashflowSummary(p.ctx, start, end); return err })
	p.check("cashflow/GetCashflowCategories", func() error { _, err := p.svc.GetCashflowCategories(p.ctx, start, end); return err })
	p.check("cashflow/GetCashflowMerchants", func() error { _, err := p.svc.GetCashflowMerchants(p.ctx, start, end); return err })
	p.check("cashflow/GetCashflowTrends", func() error {
		_, err := p.svc.GetCashflowTrends(p.ctx, &CashflowTrendOptions{StartDate: start, EndDate: end, GroupBy: "category", Period: "month"})
		return err
	})
	p.duplicates()
}

func (p *liveProbe) categories() {
	p.check("categories/ListCategories", func() error { _, err := p.svc.ListCategories(p.ctx); return err })
	p.check("categories/ListCategoryGroups", func() error { _, err := p.svc.ListCategoryGroups(p.ctx); return err })

	categories, err := p.svc.ListCategories(p.ctx)
	if err != nil || len(categories) == 0 {
		return
	}
	p.check("categories/GetCategory", func() error { _, err := p.svc.GetCategory(p.ctx, categories[0].ID); return err })
	p.check("categories/GetCategoryRollover", func() error { _, err := p.svc.GetCategoryRollover(p.ctx, categories[0].ID); return err })
}

func (p *liveProbe) goals() {
	start, end := p.monthStart(), p.today()
	p.check("goals/ListGoals", func() error { _, err := p.svc.ListGoals(p.ctx); return err })
	p.check("goals/ListSavingsGoalBudgets", func() error { _, err := p.svc.ListSavingsGoalBudgets(p.ctx, start, end); return err })

	goals, err := p.svc.ListGoals(p.ctx)
	if err != nil || len(goals) == 0 {
		if err == nil {
			p.t.Log("no goals; skipping goal-scoped probes")
		}
		return
	}
	id := goals[0].ID
	p.check("goals/GetGoal", func() error { _, err := p.svc.GetGoal(p.ctx, id); return err })
	p.check("goals/ListGoalEvents", func() error { _, err := p.svc.ListGoalEvents(p.ctx, id); return err })
	p.check("goals/GetGoalBudgetAmounts", func() error { _, err := p.svc.GetGoalBudgetAmounts(p.ctx, id, start, end); return err })
	p.check("goals/GetGoalContributions", func() error { _, err := p.svc.GetGoalContributions(p.ctx, id, start, end); return err })
}

func (p *liveProbe) investments() {
	var portfolio *InvestmentPortfolio
	p.check("investments/GetInvestmentPortfolio", func() error {
		var err error
		portfolio, err = p.svc.GetInvestmentPortfolio(p.ctx, InvestmentPortfolioOptions{})
		return err
	})

	p.check("investments/ListInvestmentAccounts", func() error { _, err := p.svc.ListInvestmentAccounts(p.ctx); return err })
	p.check("investments/ListHoldingDetails", func() error { _, err := p.svc.ListHoldingDetails(p.ctx, nil); return err })
	p.check("investments/SearchSecurities", func() error { _, err := p.svc.SearchSecurities(p.ctx, "AAPL", 5); return err })

	securityIDs := portfolioSecurityIDs(portfolio)
	if len(securityIDs) == 0 {
		p.t.Log("no securities in portfolio; skipping performance probe")
		return
	}
	start := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	p.check("investments/GetSecurityPerformance", func() error {
		_, err := p.svc.GetSecurityPerformance(p.ctx, SecurityPerformanceOptions{SecurityIDs: securityIDs[:1], StartDate: start, EndDate: p.today()})
		return err
	})
}

func (p *liveProbe) recurring() {
	start, end := p.monthStart(), p.today()
	p.check("recurring/ListRecurring", func() error { _, err := p.svc.ListRecurring(p.ctx, start, end); return err })
	p.check("recurring/ListRecurringItems", func() error { _, err := p.svc.ListRecurringItems(p.ctx, start, end); return err })
	p.check("recurring/ListRecurringStreams", func() error { _, err := p.svc.ListRecurringStreams(p.ctx); return err })
	p.check("recurring/GetRecurringSummary", func() error { _, err := p.svc.GetRecurringSummary(p.ctx, start, end); return err })
}

func (p *liveProbe) tags() {
	var tags []Tag
	p.check("tags/ListTags", func() error {
		var err error
		tags, err = p.svc.ListTags(p.ctx, "", 0)
		return err
	})
	if len(tags) == 0 {
		p.t.Log("no tags; skipping tag-scoped probes")
		return
	}
	p.check("tags/GetTag", func() error { _, err := p.svc.GetTag(p.ctx, tags[0].ID); return err })
}

func (p *liveProbe) institutions() {
	p.check("institutions/ListInstitutions", func() error { _, err := p.svc.ListInstitutions(p.ctx); return err })
}

func (p *liveProbe) credit() {
	p.check("credit/GetCreditHistory", func() error { _, err := p.svc.GetCreditHistory(p.ctx); return err })
}

func (p *liveProbe) subscription() {
	p.check("subscription/GetSubscriptionDetails", func() error { _, err := p.svc.GetSubscriptionDetails(p.ctx); return err })
}

func (p *liveProbe) household() {
	p.check("household/GetHousehold", func() error { _, err := p.svc.GetHousehold(p.ctx); return err })
	p.check("household/ListHouseholdMembers", func() error { _, err := p.svc.ListHouseholdMembers(p.ctx); return err })
	p.check("household/GetCurrentUser", func() error { _, err := p.svc.GetCurrentUser(p.ctx); return err })
	p.check("household/GetHouseholdPreferences", func() error { _, err := p.svc.GetHouseholdPreferences(p.ctx); return err })
}

func (p *liveProbe) reports() {
	p.check("reports/ListSavedReports", func() error { _, err := p.svc.ListSavedReports(p.ctx); return err })
	p.check("reports/GetReportData", func() error {
		_, err := p.svc.GetReportData(p.ctx, &ReportDataOptions{StartDate: p.monthStart(), EndDate: p.today(), GroupBy: "category"})
		return err
	})
}

func (p *liveProbe) debt() {
	p.check("debt/GetDebtPaydown", func() error { _, err := p.svc.GetDebtPaydown(p.ctx, "planned"); return err })
}

func (p *liveProbe) syncHealth() {
	p.check("institutions/GetConnectionHealth", func() error { _, err := p.svc.GetConnectionHealth(p.ctx, 3); return err })
}

func (p *liveProbe) whoami() {
	p.check("household/GetWhoAmI", func() error { _, err := p.svc.GetWhoAmI(p.ctx); return err })
}

func (p *liveProbe) merchants() {
	var merchants []*Merchant
	p.check("merchants/ListMerchants", func() error {
		var err error
		merchants, err = p.svc.ListMerchants(p.ctx, "", 5, 0, "")
		return err
	})
	if len(merchants) == 0 {
		p.t.Log("no merchants; skipping merchant-scoped probes")
		return
	}
	p.check("merchants/GetMerchant", func() error { _, err := p.svc.GetMerchant(p.ctx, merchants[0].ID); return err })
}

func (p *liveProbe) receipts() {
	var receipts []*Receipt
	p.check("receipts/ListReceipts", func() error {
		var err error
		receipts, _, err = p.svc.ListReceipts(p.ctx, "", "", 5, 0, nil)
		return err
	})
	if len(receipts) == 0 {
		p.t.Log("no receipts; skipping receipt-scoped probes")
	} else {
		id := receipts[0].ID
		p.check("receipts/GetReceipt", func() error { _, err := p.svc.GetReceipt(p.ctx, id); return err })
	}
	p.check("receipts/GetReceiptSettings", func() error { _, err := p.svc.GetReceiptSettings(p.ctx); return err })
}

func (p *liveProbe) ruleRoundtrip() {
	input := &CreateRuleInput{
		MerchantOperator: "contains",
		MerchantValue:    fmt.Sprintf("monarch-cli-live-probe-%d", time.Now().UnixNano()),
		SetCategoryID:    "",
	}
	p.t.Run("writes/rules_create_delete_roundtrip", func(t *testing.T) {
		if err := p.svc.CreateRule(p.ctx, input); err != nil {
			t.Errorf("CreateRule failed: %v", err)
			return
		}
		rules, err := p.svc.ListRules(p.ctx)
		if err != nil {
			t.Errorf("ListRules after create failed: %v", err)
			return
		}
		created := ""
		for i := range rules {
			if len(rules[i].MerchantNameCriteria) > 0 && rules[i].MerchantNameCriteria[0].Value == input.MerchantValue {
				created = rules[i].ID
				break
			}
		}
		if created == "" {
			t.Errorf("created rule %q not found in ListRules", input.MerchantValue)
			return
		}
		if err := p.svc.DeleteRule(p.ctx, created); err != nil {
			t.Errorf("DeleteRule failed: %v", err)
		}
	})
}

func (p *liveProbe) ruleReorderNoop() {
	rules, err := p.svc.ListRules(p.ctx)
	if err != nil || len(rules) == 0 {
		p.t.Log("no rules; skipping reorder probe")
		return
	}
	p.t.Run("writes/rules_reorder_noop", func(t *testing.T) {
		if _, err := p.svc.ReorderRule(p.ctx, rules[0].ID, rules[0].Order); err != nil {
			t.Errorf("ReorderRule noop failed: %v", err)
		}
	})
}

func (p *liveProbe) balanceHistoryUpload() {
	accounts, err := p.svc.ListAccounts(p.ctx)
	if err != nil || len(accounts) == 0 {
		p.t.Log("no accounts; skipping balance history upload probe")
		return
	}
	csv := fmt.Sprintf("Date,Amount,Account Name\n%s,0,%s\n", p.today(), accounts[0].DisplayName)
	p.t.Run("writes/accounts_UploadAccountBalanceHistory", func(t *testing.T) {
		if err := p.svc.UploadAccountBalanceHistory(p.ctx, accounts[0].ID, strings.NewReader(csv)); err != nil {
			t.Errorf("UploadAccountBalanceHistory failed: %v", err)
		}
	})
}

func (p *liveProbe) attachmentUpload() {
	txs, _, err := p.svc.ListTransactions(p.ctx, &ListTransactionsOptions{Limit: 1})
	if err != nil || len(txs) == 0 {
		p.t.Log("no transactions; skipping attachment upload probe")
		return
	}
	file := filepath.Join(p.t.TempDir(), "probe.png")
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89,
	}
	if err := os.WriteFile(file, png, 0o600); err != nil {
		p.t.Fatalf("write probe png: %v", err)
	}
	p.t.Run("writes/attachments_UploadAttachment", func(t *testing.T) {
		if err := p.svc.UploadAttachment(p.ctx, txs[0].ID, file); err != nil {
			t.Errorf("UploadAttachment failed: %v", err)
		}
	})
}

func (p *liveProbe) receiptUpload() {
	file := filepath.Join(p.t.TempDir(), "probe.jpg")
	if err := os.WriteFile(file, []byte("\xff\xd8\xff\xe0\njfif-probe\n"), 0o600); err != nil {
		p.t.Fatalf("write probe jpg: %v", err)
	}
	p.t.Run("writes/receipts_UploadReceiptToInbox", func(t *testing.T) {
		if _, err := p.svc.UploadReceiptToInbox(p.ctx, file); err != nil {
			t.Errorf("UploadReceiptToInbox failed: %v", err)
		}
	})
}

func portfolioSecurityIDs(p *InvestmentPortfolio) []string {
	if p == nil {
		return nil
	}
	ids := make([]string, 0, len(p.Holdings))
	for _, h := range p.Holdings {
		if h.Security.ID != "" {
			ids = append(ids, h.Security.ID)
		}
	}
	return ids
}
