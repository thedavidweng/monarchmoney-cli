package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/analyze"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

var (
	analyzeAnomaliesMonth string
	analyzeHistoryMonths  int
	analyzeMinRatio       float64
	analyzeMinAmount      float64
	analyzePastDays       int
	analyzeFutureDays     int
	analyzeMerchantsMonth string
	analyzeCompare        string
	analyzeLimit          int
	analyzeBurnRateMonth  string
)

var analyzeCmd = &cobra.Command{
	Use:     "analyze",
	Short:   "Run deterministic financial analyses",
	GroupID: "analysis",
	Long: `Run deterministic financial analyses for agent workflows.

These commands do programmatic aggregation, ratios, comparisons, and stable JSON
shaping. They do not use AI, make subjective recommendations, or mutate data.`,
	Example: `  monarch analyze anomalies --month 2026-05 --json
  monarch analyze subscriptions --json
  monarch analyze merchants --compare previous-month --limit 10 --json
  monarch analyze burn-rate --month 2026-05 --json`,
}

var analyzeAnomaliesCmd = &cobra.Command{
	Use:   "anomalies",
	Short: "Find category spending anomalies",
	Long: `Find category spending anomalies by comparing one month of expenses
against prior full-month category averages.

This command pages through transactions and performs deterministic aggregation
locally so agents do not need to group transactions themselves.`,
	Example: `  monarch analyze anomalies --json
  monarch analyze anomalies --month 2026-05 --history-months 6 --min-ratio 1.5 --min-amount 100 --json`,
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		month := normalizeAnalyzeMonth(analyzeAnomaliesMonth, start)
		if _, err := time.Parse("2006-01", month); err != nil {
			handleError(renderer, "analyze.anomalies", errors.New(errors.InvalidArguments, "--month must use YYYY-MM", errors.CatValidation, false, err), start)
			return
		}
		if analyzeHistoryMonths <= 0 {
			handleError(renderer, "analyze.anomalies", errors.New(errors.InvalidArguments, "--history-months must be greater than zero", errors.CatValidation, false, nil), start)
			return
		}

		deps, ok := newDeps(renderer, "analyze.anomalies", start)
		if !ok {
			return
		}
		svc := deps.Service

		currentStart, currentEnd, historyStart, err := analyze.AnomalyWindow(month, analyzeHistoryMonths)
		if err != nil {
			handleError(renderer, "analyze.anomalies", errors.New(errors.InvalidArguments, "--month must use YYYY-MM", errors.CatValidation, false, err), start)
			return
		}
		txs, err := svc.ListAllTransactions(cmd.Context(), monarch.ListTransactionsOptions{Limit: 1000, StartDate: historyStart, EndDate: currentEnd})
		if err != nil {
			handleError(renderer, "analyze.anomalies", wrapError(err, "failed to list transactions"), start)
			return
		}
		result, err := analyze.BuildAnomalies(txs, analyze.AnomalyOptions{
			Month:         strings.TrimSuffix(currentStart, "-01"),
			HistoryMonths: analyzeHistoryMonths,
			MinRatio:      analyzeMinRatio,
			MinAmount:     analyzeMinAmount,
		})
		if err != nil {
			handleError(renderer, "analyze.anomalies", errors.New(errors.InvalidArguments, "failed to analyze anomalies", errors.CatValidation, false, err), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("analyze.anomalies", profile, output.SchemaVersion, "", map[string]interface{}{"period": map[string]string{"start_date": currentStart, "end_date": currentEnd}, "anomalies": result}, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}
		fmt.Printf("%-30s %12s %12s %8s %-8s %-20s %12s\n", "CATEGORY", "CURRENT", "AVG", "RATIO", "SEVERITY", "LARGEST MERCHANT", "AMOUNT")
		for _, a := range result {
			fmt.Printf("%-30s %12.2f %12.2f %8.2f %-8s %-20s %12.2f\n", a.Category, a.CurrentMonth, a.AvgHistory, a.Ratio, a.Severity, a.LargestMerchant, a.LargestAmount)
		}
	},
}

var analyzeSubscriptionsCmd = &cobra.Command{
	Use:   "subscriptions",
	Short: "Summarize recurring subscriptions",
	Long: `Summarize recurring subscriptions from Monarch recurring items.

The output includes monthly and annualized amounts, last and next charges, and
potential overlap facts. Overlaps are facts for agent review, not judgments that
the services are wasteful.`,
	Example: `  monarch analyze subscriptions --json
  monarch analyze subscriptions --past-days 370 --future-days 370 --json`,
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		if analyzePastDays < 0 || analyzeFutureDays < 0 {
			handleError(renderer, "analyze.subscriptions", errors.New(errors.InvalidArguments, "day windows must be non-negative", errors.CatValidation, false, nil), start)
			return
		}

		deps, ok := newDeps(renderer, "analyze.subscriptions", start)
		if !ok {
			return
		}
		svc := deps.Service
		startDate := start.AddDate(0, 0, -analyzePastDays).Format("2006-01-02")
		endDate := start.AddDate(0, 0, analyzeFutureDays).Format("2006-01-02")
		items, err := svc.ListRecurringItems(cmd.Context(), startDate, endDate)
		if err != nil {
			handleError(renderer, "analyze.subscriptions", wrapError(err, "failed to list recurring items"), start)
			return
		}
		result := analyze.BuildSubscriptions(items)
		if jsonMode {
			env := output.NewEnvelope("analyze.subscriptions", profile, output.SchemaVersion, "", result, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}
		fmt.Printf("%-24s %10s %10s %-12s %-12s %-12s %s\n", "MERCHANT", "MONTHLY", "ANNUAL", "FREQUENCY", "LAST", "NEXT", "CATEGORY")
		for _, sub := range result.Subscriptions {
			fmt.Printf("%-24s %10.2f %10.2f %-12s %-12s %-12s %s\n", sub.Merchant, sub.Monthly, sub.Annual, sub.Frequency, sub.LastCharge, sub.NextCharge, sub.Category)
		}
	},
}

var analyzeMerchantsCmd = &cobra.Command{
	Use:   "merchants",
	Short: "Compare merchant spending",
	Long: `Compare merchant expenses between the selected month and a previous period.

The v1 comparison mode is previous-month. The command returns expense_current,
expense_previous, change_pct, and direction with stable semantics for agents.`,
	Example: `  monarch analyze merchants --compare previous-month --json
  monarch analyze merchants --month 2026-05 --compare previous-month --limit 20 --json`,
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		if analyzeCompare != "previous-month" {
			handleError(renderer, "analyze.merchants", errors.New(errors.InvalidArguments, "--compare currently supports previous-month", errors.CatValidation, false, nil), start)
			return
		}
		month := normalizeAnalyzeMonth(analyzeMerchantsMonth, start)
		current, previous, err := analyze.PreviousMonthComparisonWindow(month)
		if err != nil {
			handleError(renderer, "analyze.merchants", errors.New(errors.InvalidArguments, "--month must use YYYY-MM", errors.CatValidation, false, err), start)
			return
		}
		deps, ok := newDeps(renderer, "analyze.merchants", start)
		if !ok {
			return
		}
		svc := deps.Service
		currentRecords, err := svc.GetCashflowMerchants(cmd.Context(), current.StartDate, current.EndDate)
		if err != nil {
			handleError(renderer, "analyze.merchants", wrapError(err, "failed to get current merchant spending"), start)
			return
		}
		previousRecords, err := svc.GetCashflowMerchants(cmd.Context(), previous.StartDate, previous.EndDate)
		if err != nil {
			handleError(renderer, "analyze.merchants", wrapError(err, "failed to get previous merchant spending"), start)
			return
		}
		result := analyze.BuildMerchantComparison(currentRecords, previousRecords, analyzeLimit)
		if jsonMode {
			env := output.NewEnvelope("analyze.merchants", profile, output.SchemaVersion, "", map[string]interface{}{"period": current, "previous_period": previous, "comparison": result}, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}
		fmt.Printf("%-24s %12s %12s %12s %s\n", "MERCHANT", "CURRENT", "PREVIOUS", "CHANGE %", "DIRECTION")
		for _, row := range result {
			change := "null"
			if row.ChangePct != nil {
				change = fmt.Sprintf("%.2f", *row.ChangePct)
			}
			fmt.Printf("%-24s %12.2f %12.2f %12s %s\n", row.Merchant, row.ExpenseCurrent, row.ExpensePrevious, change, row.Direction)
		}
	},
}

var analyzeBurnRateCmd = &cobra.Command{
	Use:   "burn-rate",
	Short: "Compare budget usage with elapsed month time",
	Long: `Compare current budget usage with elapsed month time.

This command uses Monarch budget actual/planned values and deterministic date
math. It does not re-sum transactions or make subjective budget advice.`,
	Example: `  monarch analyze burn-rate --json
  monarch analyze burn-rate --month 2026-05 --json`,
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		month := normalizeAnalyzeMonth(analyzeBurnRateMonth, start)
		monthStart, monthEnd, err := monthRange(month)
		if err != nil {
			handleError(renderer, "analyze.burn-rate", errors.New(errors.InvalidArguments, "--month must use YYYY-MM", errors.CatValidation, false, err), start)
			return
		}
		deps, ok := newDeps(renderer, "analyze.burn-rate", start)
		if !ok {
			return
		}
		svc := deps.Service
		budgets, err := svc.ListBudgets(cmd.Context(), monarch.ListBudgetsOptions{StartDate: monthStart, EndDate: monthEnd})
		if err != nil {
			handleError(renderer, "analyze.burn-rate", wrapError(err, "failed to list budgets"), start)
			return
		}
		now := start
		if month != start.Format("2006-01") {
			parsed, _ := time.Parse("2006-01", month)
			now = time.Date(parsed.Year(), parsed.Month(), time.Date(parsed.Year(), parsed.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day(), 12, 0, 0, 0, time.UTC)
		}
		result, err := analyze.BuildBurnRate(budgets, now)
		if err != nil {
			handleError(renderer, "analyze.burn-rate", errors.New(errors.InternalError, "failed to analyze burn rate", errors.CatInternal, false, err), start)
			return
		}
		if jsonMode {
			env := output.NewEnvelope("analyze.burn-rate", profile, output.SchemaVersion, "", map[string]interface{}{"period": map[string]string{"start_date": monthStart, "end_date": monthEnd}, "budgets": result}, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}
		fmt.Printf("%-30s %10s %10s %10s %8s %8s %s\n", "CATEGORY", "BUDGETED", "SPENT", "REMAINING", "BURN %", "TIME %", "STATUS")
		for _, b := range result {
			fmt.Printf("%-30s %10.2f %10.2f %10.2f %8.2f %8.2f %s\n", b.Category, b.Budgeted, b.Spent, b.Remaining, b.BurnPct, b.TimePct, b.Status)
		}
	},
}

func normalizeAnalyzeMonth(value string, now time.Time) string {
	if value != "" {
		return value
	}
	return now.Format("2006-01")
}

func monthRange(month string) (string, string, error) {
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return "", "", err
	}
	end := time.Date(start.Year(), start.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	return start.Format("2006-01-02"), end.Format("2006-01-02"), nil
}

func init() {
	analyzeAnomaliesCmd.Flags().StringVar(&analyzeAnomaliesMonth, "month", "", "month to analyze (YYYY-MM)")
	analyzeAnomaliesCmd.Flags().IntVar(&analyzeHistoryMonths, "history-months", 6, "number of prior full months to compare")
	analyzeAnomaliesCmd.Flags().Float64Var(&analyzeMinRatio, "min-ratio", 1.5, "minimum current/history ratio")
	analyzeAnomaliesCmd.Flags().Float64Var(&analyzeMinAmount, "min-amount", 100, "minimum current month expense")

	analyzeSubscriptionsCmd.Flags().IntVar(&analyzePastDays, "past-days", 370, "days in the past to inspect")
	analyzeSubscriptionsCmd.Flags().IntVar(&analyzeFutureDays, "future-days", 370, "days in the future to inspect")

	analyzeMerchantsCmd.Flags().StringVar(&analyzeCompare, "compare", "previous-month", "comparison period (previous-month)")
	_ = analyzeMerchantsCmd.RegisterFlagCompletionFunc("compare", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"previous-month", "previous-quarter", "previous-year"}, cobra.ShellCompDirectiveNoFileComp
	})
	analyzeMerchantsCmd.Flags().StringVar(&analyzeMerchantsMonth, "month", "", "month to analyze (YYYY-MM)")
	analyzeMerchantsCmd.Flags().IntVar(&analyzeLimit, "limit", 20, "maximum merchants to return")

	analyzeBurnRateCmd.Flags().StringVar(&analyzeBurnRateMonth, "month", "", "month to analyze (YYYY-MM)")

	analyzeCmd.AddCommand(analyzeAnomaliesCmd)
	analyzeCmd.AddCommand(analyzeSubscriptionsCmd)
	analyzeCmd.AddCommand(analyzeMerchantsCmd)
	analyzeCmd.AddCommand(analyzeBurnRateCmd)
	RootCmd.AddCommand(analyzeCmd)
}
