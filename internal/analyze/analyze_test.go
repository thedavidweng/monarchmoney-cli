package analyze

import (
	"reflect"
	"testing"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
)

func TestBuildAnomaliesUsesExpensesAndStableSeverity(t *testing.T) {
	txs := []monarch.Transaction{
		{ID: "h1", Date: "2025-11-15", Amount: -100, Category: "Dining", Merchant: "Cafe"},
		{ID: "h2", Date: "2025-12-15", Amount: -100, Category: "Dining", Merchant: "Cafe"},
		{ID: "h3", Date: "2026-01-15", Amount: -100, Category: "Dining", Merchant: "Cafe"},
		{ID: "h4", Date: "2026-02-15", Amount: -100, Category: "Dining", Merchant: "Cafe"},
		{ID: "h5", Date: "2026-03-15", Amount: -100, Category: "Dining", Merchant: "Cafe"},
		{ID: "h6", Date: "2026-04-15", Amount: -100, Category: "Dining", Merchant: "Cafe"},
		{ID: "income", Date: "2026-05-01", Amount: 1000, Category: "Dining", Merchant: "Paycheck"},
		{ID: "c1", Date: "2026-05-02", Amount: -150, Category: "Dining", Merchant: "Restaurant"},
		{ID: "c2", Date: "2026-05-03", Amount: -70, Category: "Dining", Merchant: "Uber Eats"},
		{ID: "c3", Date: "2026-05-04", Amount: -80, Category: "Dining", Merchant: "Uber Eats"},
	}

	got, err := BuildAnomalies(txs, AnomalyOptions{
		Month:         "2026-05",
		HistoryMonths: 6,
		MinRatio:      1.5,
		MinAmount:     100,
	})
	if err != nil {
		t.Fatalf("BuildAnomalies() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("BuildAnomalies() len = %d, want 1: %#v", len(got), got)
	}
	a := got[0]
	if a.Category != "Dining" || a.CurrentMonth != 300 || a.AvgHistory != 100 || a.Ratio != 3 || a.Severity != "high" {
		t.Fatalf("anomaly = %#v", a)
	}
	if a.LargestMerchant != "Restaurant" || a.LargestAmount != 150 {
		t.Fatalf("largest merchant = %q %.2f, want Restaurant 150", a.LargestMerchant, a.LargestAmount)
	}
}

func TestBuildAnomaliesSkipsZeroHistoryAverage(t *testing.T) {
	got, err := BuildAnomalies([]monarch.Transaction{
		{ID: "new", Date: "2026-05-02", Amount: -250, Category: "Travel", Merchant: "Airline"},
	}, AnomalyOptions{Month: "2026-05", HistoryMonths: 6, MinRatio: 1.5, MinAmount: 100})
	if err != nil {
		t.Fatalf("BuildAnomalies() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("BuildAnomalies() = %#v, want no ratio-based anomaly with zero history", got)
	}
}

func TestBuildMerchantComparisonHandlesPreviousZero(t *testing.T) {
	got := BuildMerchantComparison(
		[]monarch.CashflowRecord{{Name: "Amazon", Amount: 320}, {Name: "Cafe", Amount: 50}},
		[]monarch.CashflowRecord{{Name: "Cafe", Amount: 100}},
		10,
	)

	if len(got) != 2 {
		t.Fatalf("BuildMerchantComparison() len = %d, want 2", len(got))
	}
	if got[0].Merchant != "Amazon" || got[0].ExpenseCurrent != 320 || got[0].ExpensePrevious != 0 || got[0].ChangePct != nil || got[0].Direction != "new" {
		t.Fatalf("new merchant comparison = %#v", got[0])
	}
	if got[1].Merchant != "Cafe" || got[1].Direction != "down" || got[1].ChangePct == nil || *got[1].ChangePct != -50 {
		t.Fatalf("existing merchant comparison = %#v", got[1])
	}
}

func TestBuildSubscriptionsDedupesAndComputesAnnualAmounts(t *testing.T) {
	items := []monarch.RecurringItem{
		{Stream: monarch.RecurringStream{ID: "netflix", Frequency: "monthly", Amount: 15.49, MerchantName: "Netflix"}, Date: "2026-04-01", IsPast: true, CategoryName: "Entertainment"},
		{Stream: monarch.RecurringStream{ID: "netflix", Frequency: "monthly", Amount: 15.49, MerchantName: "Netflix"}, Date: "2026-05-01", IsPast: false, CategoryName: "Entertainment"},
		{Stream: monarch.RecurringStream{ID: "prime", Frequency: "yearly", Amount: 139, MerchantName: "Amazon Prime", IsApproximate: true}, Date: "2026-02-01", IsPast: true, CategoryName: "Shopping"},
	}

	got := BuildSubscriptions(items)

	if len(got.Subscriptions) != 2 {
		t.Fatalf("subscriptions len = %d, want 2: %#v", len(got.Subscriptions), got.Subscriptions)
	}
	if got.Subscriptions[0].Merchant != "Amazon Prime" || got.Subscriptions[0].Monthly != 11.58 || got.Subscriptions[0].Annual != 139 || !got.Subscriptions[0].IsApproximate {
		t.Fatalf("annual subscription = %#v", got.Subscriptions[0])
	}
	if got.Subscriptions[1].Merchant != "Netflix" || got.Subscriptions[1].Monthly != 15.49 || got.Subscriptions[1].Annual != 185.88 || got.Subscriptions[1].LastCharge != "2026-04-01" || got.Subscriptions[1].NextCharge != "2026-05-01" {
		t.Fatalf("monthly subscription = %#v", got.Subscriptions[1])
	}
	if got.TotalMonthly != 27.07 || got.TotalAnnual != 324.88 {
		t.Fatalf("totals = %.2f %.2f", got.TotalMonthly, got.TotalAnnual)
	}
}

func TestBuildBurnRateUsesBudgetAndElapsedTime(t *testing.T) {
	got, err := BuildBurnRate([]monarch.Budget{
		{CategoryName: "Dining", Planned: 600, Actual: 552},
		{CategoryName: "Books", Planned: 100, Actual: 10},
	}, time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("BuildBurnRate() error = %v", err)
	}

	wantStatuses := []string{"overspending", "underused"}
	var statuses []string
	for _, b := range got {
		statuses = append(statuses, b.Status)
	}
	if !reflect.DeepEqual(statuses, wantStatuses) {
		t.Fatalf("statuses = %#v, want %#v; budgets=%#v", statuses, wantStatuses, got)
	}
	if got[0].Budgeted != 600 || got[0].Spent != 552 || got[0].Remaining != 48 || got[0].BurnPct != 92 || got[0].TimePct != 77.42 {
		t.Fatalf("burn-rate budget = %#v", got[0])
	}
}

func TestDateWindows(t *testing.T) {
	currentStart, currentEnd, historyStart, err := AnomalyWindow("2026-05", 6)
	if err != nil {
		t.Fatalf("AnomalyWindow() error = %v", err)
	}
	if currentStart != "2026-05-01" || currentEnd != "2026-05-31" || historyStart != "2025-11-01" {
		t.Fatalf("AnomalyWindow() = %s %s %s", currentStart, currentEnd, historyStart)
	}

	current, previous, err := PreviousMonthComparisonWindow("2026-05")
	if err != nil {
		t.Fatalf("PreviousMonthComparisonWindow() error = %v", err)
	}
	if current.StartDate != "2026-05-01" || current.EndDate != "2026-05-31" || previous.StartDate != "2026-04-01" || previous.EndDate != "2026-04-30" {
		t.Fatalf("PreviousMonthComparisonWindow() = %#v %#v", current, previous)
	}
}
