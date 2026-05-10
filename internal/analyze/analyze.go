package analyze

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
)

type AnomalyOptions struct {
	Month         string
	HistoryMonths int
	MinRatio      float64
	MinAmount     float64
}

type Anomaly struct {
	Category        string  `json:"category"`
	CurrentMonth    float64 `json:"current_month"`
	AvgHistory      float64 `json:"avg_history"`
	Ratio           float64 `json:"ratio"`
	Severity        string  `json:"severity"`
	LargestMerchant string  `json:"largest_merchant"`
	LargestAmount   float64 `json:"largest_amount"`
}

type MerchantComparison struct {
	Merchant        string   `json:"merchant"`
	ExpenseCurrent  float64  `json:"expense_current"`
	ExpensePrevious float64  `json:"expense_previous"`
	ChangePct       *float64 `json:"change_pct"`
	Direction       string   `json:"direction"`
}

type SubscriptionSummary struct {
	Subscriptions     []Subscription     `json:"subscriptions"`
	TotalMonthly      float64            `json:"total_monthly"`
	TotalAnnual       float64            `json:"total_annual"`
	PotentialOverlaps []PotentialOverlap `json:"potential_overlaps"`
}

type Subscription struct {
	Merchant      string  `json:"merchant"`
	Monthly       float64 `json:"monthly"`
	Annual        float64 `json:"annual"`
	Frequency     string  `json:"frequency"`
	LastCharge    string  `json:"last_charge,omitempty"`
	NextCharge    string  `json:"next_charge,omitempty"`
	Category      string  `json:"category"`
	IsApproximate bool    `json:"is_approximate"`
}

type PotentialOverlap struct {
	Group          string   `json:"group"`
	Services       []string `json:"services"`
	CombinedAnnual float64  `json:"combined_annual"`
}

type BurnRateBudget struct {
	Category    string  `json:"category"`
	Budgeted    float64 `json:"budgeted"`
	Spent       float64 `json:"spent"`
	Remaining   float64 `json:"remaining"`
	DaysElapsed int     `json:"days_elapsed"`
	DaysTotal   int     `json:"days_total"`
	BurnPct     float64 `json:"burn_pct"`
	TimePct     float64 `json:"time_pct"`
	Status      string  `json:"status"`
}

type DateRange struct {
	StartDate string
	EndDate   string
}

func AnomalyWindow(month string, historyMonths int) (currentStart string, currentEnd string, historyStart string, err error) {
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return "", "", "", err
	}
	if historyMonths <= 0 {
		historyMonths = 6
	}
	end := time.Date(start.Year(), start.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	return start.Format("2006-01-02"), end.Format("2006-01-02"), start.AddDate(0, -historyMonths, 0).Format("2006-01-02"), nil
}

func PreviousMonthComparisonWindow(month string) (DateRange, DateRange, error) {
	currentStart, err := time.Parse("2006-01", month)
	if err != nil {
		return DateRange{}, DateRange{}, err
	}
	currentEnd := time.Date(currentStart.Year(), currentStart.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	previousStart := currentStart.AddDate(0, -1, 0)
	previousEnd := time.Date(previousStart.Year(), previousStart.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	return DateRange{StartDate: currentStart.Format("2006-01-02"), EndDate: currentEnd.Format("2006-01-02")},
		DateRange{StartDate: previousStart.Format("2006-01-02"), EndDate: previousEnd.Format("2006-01-02")},
		nil
}

func BuildAnomalies(txs []monarch.Transaction, opts AnomalyOptions) ([]Anomaly, error) {
	currentStart, err := time.Parse("2006-01", opts.Month)
	if err != nil {
		return nil, err
	}
	opts = normalizeAnomalyOptions(opts)
	currentKey := currentStart.Format("2006-01")

	current, history, merchantTotals, err := aggregateAnomalyData(txs, currentStart, currentKey, opts.HistoryMonths)
	if err != nil {
		return nil, err
	}

	return buildAnomalyList(current, history, merchantTotals, opts), nil
}

func normalizeAnomalyOptions(opts AnomalyOptions) AnomalyOptions {
	if opts.HistoryMonths <= 0 {
		opts.HistoryMonths = 6
	}
	if opts.MinRatio <= 0 {
		opts.MinRatio = 1.5
	}
	return opts
}

func aggregateAnomalyData(txs []monarch.Transaction, currentStart time.Time, currentKey string, historyMonths int) (map[string]float64, map[string]map[string]float64, map[string]map[string]float64, error) {
	history := make(map[string]map[string]float64)
	current := make(map[string]float64)
	merchantTotals := make(map[string]map[string]float64)
	historyStart := currentStart.AddDate(0, -historyMonths, 0)

	for _, tx := range txs {
		if tx.Amount >= 0 || tx.Category == "" {
			continue
		}

		day, err := time.Parse("2006-01-02", tx.Date)
		if err != nil {
			return nil, nil, nil, err
		}
		monthKey := day.Format("2006-01")
		amount := round2(math.Abs(tx.Amount))

		if monthKey == currentKey {
			current[tx.Category] += amount
			addMerchantTotal(merchantTotals, tx.Category, tx.Merchant, amount)
			continue
		}
		if !day.Before(currentStart) || day.Before(historyStart) {
			continue
		}
		addHistoryTotal(history, tx.Category, monthKey, amount)
	}

	return current, history, merchantTotals, nil
}

func addMerchantTotal(merchantTotals map[string]map[string]float64, category, merchant string, amount float64) {
	if merchantTotals[category] == nil {
		merchantTotals[category] = make(map[string]float64)
	}
	merchantTotals[category][merchant] += amount
}

func addHistoryTotal(history map[string]map[string]float64, category, monthKey string, amount float64) {
	if history[category] == nil {
		history[category] = make(map[string]float64)
	}
	history[category][monthKey] += amount
}

func buildAnomalyList(current map[string]float64, history map[string]map[string]float64, merchantTotals map[string]map[string]float64, opts AnomalyOptions) []Anomaly {
	out := make([]Anomaly, 0)
	for category, total := range current {
		if total < opts.MinAmount {
			continue
		}
		avg := averageHistory(history[category], opts.HistoryMonths)
		if avg == 0 {
			continue
		}
		ratio := round2(total / avg)
		if ratio < opts.MinRatio {
			continue
		}
		merchant, merchantAmount := largestMerchant(merchantTotals[category])
		out = append(out, Anomaly{
			Category:        category,
			CurrentMonth:    round2(total),
			AvgHistory:      round2(avg),
			Ratio:           ratio,
			Severity:        anomalySeverity(ratio),
			LargestMerchant: merchant,
			LargestAmount:   round2(merchantAmount),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Ratio == out[j].Ratio {
			return out[i].Category < out[j].Category
		}
		return out[i].Ratio > out[j].Ratio
	})
	return out
}

func BuildMerchantComparison(current, previous []monarch.CashflowRecord, limit int) []MerchantComparison {
	previousByMerchant := make(map[string]float64, len(previous))
	for _, record := range previous {
		previousByMerchant[record.Name] = expenseValue(record.Amount)
	}

	seen := make(map[string]bool, len(current))
	out := make([]MerchantComparison, 0, len(current))
	for _, record := range current {
		merchant := record.Name
		currentExpense := expenseValue(record.Amount)
		previousExpense := previousByMerchant[merchant]
		seen[merchant] = true
		out = append(out, merchantComparison(merchant, currentExpense, previousExpense))
	}
	for _, record := range previous {
		if seen[record.Name] {
			continue
		}
		out = append(out, merchantComparison(record.Name, 0, expenseValue(record.Amount)))
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].ExpenseCurrent == out[j].ExpenseCurrent {
			return out[i].Merchant < out[j].Merchant
		}
		return out[i].ExpenseCurrent > out[j].ExpenseCurrent
	})
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func BuildSubscriptions(items []monarch.RecurringItem) SubscriptionSummary {
	byStream := make(map[string]*Subscription)
	for _, item := range items {
		id := item.Stream.ID
		if id == "" {
			id = item.Stream.MerchantName + "|" + item.Stream.Frequency
		}
		sub := byStream[id]
		if sub == nil {
			monthly, annual := subscriptionAmounts(item.Stream.Frequency, item.Stream.Amount)
			sub = &Subscription{
				Merchant:      item.Stream.MerchantName,
				Monthly:       monthly,
				Annual:        annual,
				Frequency:     item.Stream.Frequency,
				Category:      item.CategoryName,
				IsApproximate: item.Stream.IsApproximate,
			}
			byStream[id] = sub
		}
		if item.CategoryName != "" {
			sub.Category = item.CategoryName
		}
		if item.IsPast {
			if sub.LastCharge == "" || item.Date > sub.LastCharge {
				sub.LastCharge = item.Date
			}
			continue
		}
		if sub.NextCharge == "" || item.Date < sub.NextCharge {
			sub.NextCharge = item.Date
		}
	}

	out := SubscriptionSummary{}
	for _, sub := range byStream {
		out.Subscriptions = append(out.Subscriptions, *sub)
		out.TotalMonthly += sub.Monthly
		out.TotalAnnual += sub.Annual
	}
	sort.Slice(out.Subscriptions, func(i, j int) bool {
		return out.Subscriptions[i].Merchant < out.Subscriptions[j].Merchant
	})
	out.TotalMonthly = round2(out.TotalMonthly)
	out.TotalAnnual = round2(out.TotalAnnual)
	out.PotentialOverlaps = buildPotentialOverlaps(out.Subscriptions)
	return out
}

func BuildBurnRate(budgets []monarch.Budget, now time.Time) ([]BurnRateBudget, error) {
	daysTotal := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	daysElapsed := now.Day()
	timePct := round2(float64(daysElapsed) / float64(daysTotal) * 100)

	out := make([]BurnRateBudget, 0, len(budgets))
	for _, budget := range budgets {
		if budget.Planned <= 0 {
			continue
		}
		burnPct := round2(budget.Actual / budget.Planned * 100)
		out = append(out, BurnRateBudget{
			Category:    budget.CategoryName,
			Budgeted:    round2(budget.Planned),
			Spent:       round2(budget.Actual),
			Remaining:   round2(budget.Planned - budget.Actual),
			DaysElapsed: daysElapsed,
			DaysTotal:   daysTotal,
			BurnPct:     burnPct,
			TimePct:     timePct,
			Status:      burnStatus(burnPct, timePct),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].BurnPct == out[j].BurnPct {
			return out[i].Category < out[j].Category
		}
		return out[i].BurnPct > out[j].BurnPct
	})
	return out, nil
}

func averageHistory(months map[string]float64, historyMonths int) float64 {
	if historyMonths <= 0 {
		return 0
	}
	var total float64
	for _, amount := range months {
		total += amount
	}
	return total / float64(historyMonths)
}

func largestMerchant(merchants map[string]float64) (string, float64) {
	var name string
	var amount float64
	for merchant, total := range merchants {
		if total > amount || (total == amount && merchant < name) {
			name = merchant
			amount = total
		}
	}
	return name, amount
}

func anomalySeverity(ratio float64) string {
	if ratio >= 3 {
		return "high"
	}
	if ratio >= 2 {
		return "medium"
	}
	return "low"
}

func merchantComparison(merchant string, current, previous float64) MerchantComparison {
	out := MerchantComparison{
		Merchant:        merchant,
		ExpenseCurrent:  round2(current),
		ExpensePrevious: round2(previous),
	}
	switch {
	case previous == 0 && current > 0:
		out.Direction = "new"
	case current == 0 && previous > 0:
		change := -100.0
		out.ChangePct = &change
		out.Direction = "down"
	case current > previous:
		change := round2((current - previous) / previous * 100)
		out.ChangePct = &change
		out.Direction = "up"
	case current < previous:
		change := round2((current - previous) / previous * 100)
		out.ChangePct = &change
		out.Direction = "down"
	default:
		change := 0.0
		out.ChangePct = &change
		out.Direction = "flat"
	}
	return out
}

func subscriptionAmounts(frequency string, amount float64) (float64, float64) {
	switch strings.ToLower(frequency) {
	case "yearly", "annual", "annually":
		return round2(amount / 12), round2(amount)
	case "weekly":
		annual := amount * 52
		return round2(annual / 12), round2(annual)
	case "biweekly":
		annual := amount * 26
		return round2(annual / 12), round2(annual)
	default:
		return round2(amount), round2(amount * 12)
	}
}

func buildPotentialOverlaps(subs []Subscription) []PotentialOverlap {
	groups := map[string][]string{
		"Streaming": {"netflix", "disney", "hbo", "max", "hulu", "paramount", "peacock", "prime video", "apple tv"},
		"Music":     {"spotify", "apple music", "youtube music", "tidal"},
		"Cloud":     {"icloud", "dropbox", "google one", "onedrive"},
	}
	var overlaps []PotentialOverlap
	for group, needles := range groups {
		var services []string
		var annual float64
		for _, sub := range subs {
			name := strings.ToLower(sub.Merchant)
			for _, needle := range needles {
				if strings.Contains(name, needle) {
					services = append(services, sub.Merchant)
					annual += sub.Annual
					break
				}
			}
		}
		if len(services) > 1 {
			sort.Strings(services)
			overlaps = append(overlaps, PotentialOverlap{Group: group, Services: services, CombinedAnnual: round2(annual)})
		}
	}
	sort.Slice(overlaps, func(i, j int) bool {
		return overlaps[i].Group < overlaps[j].Group
	})
	return overlaps
}

func burnStatus(burnPct, timePct float64) string {
	switch {
	case burnPct >= 100 || burnPct-timePct >= 10:
		return "overspending"
	case burnPct-timePct >= 5:
		return "ahead"
	case timePct-burnPct >= 25:
		return "underused"
	default:
		return "on_track"
	}
}

func expenseValue(amount float64) float64 {
	return round2(math.Abs(amount))
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
