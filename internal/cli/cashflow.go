package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

var (
	cfStartDate        string
	cfEndDate          string
	cfTrendGroupBy     string
	cfTrendPeriod      string
	cfTrendAccountIDs  []string
	cfTrendCategoryIDs []string
)

var cashflowCmd = &cobra.Command{
	Use:   "cashflow",
	Short: "Manage Monarch Money cashflow",
}

var cashflowSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Get cashflow summary",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "cashflow.summary", start)
		if !ok {
			return
		}
		svc := deps.Service

		setCashflowDates()

		summary, err := svc.GetCashflowSummary(cmd.Context(), cfStartDate, cfEndDate)
		if err != nil {
			handleError(renderer, "cashflow.summary", wrapError(err, "failed to get cashflow summary"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("cashflow.summary", profile, output.SchemaVersion, "", summary, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Cashflow Summary (%s to %s):\n", cfStartDate, cfEndDate)
			fmt.Printf("Income:       %.2f\n", summary.Income)
			fmt.Printf("Expense:      %.2f\n", summary.Expense)
			fmt.Printf("Savings:      %.2f\n", summary.Savings)
			fmt.Printf("Savings Rate: %.2f%%\n", summary.SavingsRate*100)
		}
	},
}

var cashflowCategoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "Get cashflow by category",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "cashflow.categories", start)
		if !ok {
			return
		}
		svc := deps.Service

		setCashflowDates()

		records, err := svc.GetCashflowCategories(cmd.Context(), cfStartDate, cfEndDate)
		if err != nil {
			handleError(renderer, "cashflow.categories", wrapError(err, "failed to get cashflow categories"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("cashflow.categories", profile, output.SchemaVersion, "", records, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-30s %10s\n", "CATEGORY", "AMOUNT")
			for _, r := range records {
				fmt.Printf("%-30s %10.2f\n", r.Name, r.Amount)
			}
		}
	},
}

var cashflowMerchantsCmd = &cobra.Command{
	Use:   "merchants",
	Short: "Get cashflow by merchant",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "cashflow.merchants", start)
		if !ok {
			return
		}
		svc := deps.Service

		setCashflowDates()

		records, err := svc.GetCashflowMerchants(cmd.Context(), cfStartDate, cfEndDate)
		if err != nil {
			handleError(renderer, "cashflow.merchants", wrapError(err, "failed to get cashflow merchants"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("cashflow.merchants", profile, output.SchemaVersion, "", records, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-30s %10s\n", "MERCHANT", "AMOUNT")
			for _, r := range records {
				fmt.Printf("%-30s %10.2f\n", r.Name, r.Amount)
			}
		}
	},
}

var cashflowTrendsCmd = &cobra.Command{
	Use:   "trends",
	Short: "Get cashflow trends grouped by category or category group",
	Long:  "Get aggregate cashflow rows grouped by category or category group and bucketed by month, quarter, or year.",
	Example: `  monarch cashflow trends --from 2026-01-01 --to 2026-03-31 --group-by category --period month
  monarch cashflow trends --from 2026-01-01 --to 2026-12-31 --group-by category-group --period quarter --account-id acc_123 --json --pretty`,
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if cfStartDate == "" || cfEndDate == "" {
			handleError(renderer, "cashflow.trends", errors.New(errors.InvalidArguments, "--from and --to are required", errors.CatValidation, false, nil), start)
			return
		}
		if _, err := time.Parse("2006-01-02", cfStartDate); err != nil {
			handleError(renderer, "cashflow.trends", errors.New(errors.InvalidArguments, "from date must use YYYY-MM-DD", errors.CatValidation, false, err), start)
			return
		}
		if _, err := time.Parse("2006-01-02", cfEndDate); err != nil {
			handleError(renderer, "cashflow.trends", errors.New(errors.InvalidArguments, "to date must use YYYY-MM-DD", errors.CatValidation, false, err), start)
			return
		}
		if cfTrendGroupBy != "category" && cfTrendGroupBy != "category-group" {
			handleError(renderer, "cashflow.trends", errors.New(errors.InvalidArguments, "group-by must be category or category-group", errors.CatValidation, false, nil), start)
			return
		}
		if cfTrendPeriod != "month" && cfTrendPeriod != "quarter" && cfTrendPeriod != "year" {
			handleError(renderer, "cashflow.trends", errors.New(errors.InvalidArguments, "period must be month, quarter, or year", errors.CatValidation, false, nil), start)
			return
		}

		deps, ok := newDeps(renderer, "cashflow.trends", start)
		if !ok {
			return
		}
		svc := deps.Service

		rows, err := svc.GetCashflowTrends(cmd.Context(), monarch.CashflowTrendOptions{
			StartDate:   cfStartDate,
			EndDate:     cfEndDate,
			GroupBy:     cfTrendGroupBy,
			Period:      cfTrendPeriod,
			AccountIDs:  cfTrendAccountIDs,
			CategoryIDs: cfTrendCategoryIDs,
		})
		if err != nil {
			handleError(renderer, "cashflow.trends", wrapError(err, "failed to get cashflow trends"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("cashflow.trends", profile, output.SchemaVersion, "", rows, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-12s %-30s %12s %12s %12s\n", "PERIOD", "GROUP", "SUM", "INCOME", "EXPENSE")
			for _, row := range rows {
				group := row.GroupName
				if group == "" {
					group = row.GroupID
				}
				fmt.Printf("%-12s %-30s %12.2f %12.2f %12.2f\n", row.Period, group, row.Sum, row.SumIncome, row.SumExpense)
			}
		}
	},
}

func setCashflowDates() {
	if cfStartDate == "" {
		now := time.Now()
		cfStartDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}
	if cfEndDate == "" {
		now := time.Now()
		cfEndDate = now.Format("2006-01-02")
	}
}

var cashflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "Get cashflow records by period",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "cashflow.list", start)
		if !ok {
			return
		}
		svc := deps.Service

		setCashflowDates()

		records, err := svc.ListCashflow(cmd.Context(), cfStartDate, cfEndDate)
		if err != nil {
			handleError(renderer, "cashflow.list", wrapError(err, "failed to list cashflow"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("cashflow.list", profile, output.SchemaVersion, "", records, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-12s %10s %10s %10s\n", "PERIOD", "INCOME", "EXPENSE", "SAVINGS")
			for _, r := range records {
				fmt.Printf("%-12s %10.2f %10.2f %10.2f\n", r.Period, r.Income, r.Expense, r.Savings)
			}
		}
	},
}

func init() {
	cashflowCmd.PersistentFlags().StringVar(&cfStartDate, "from", "", "start date (YYYY-MM-DD)")
	cashflowCmd.PersistentFlags().StringVar(&cfEndDate, "to", "", "end date (YYYY-MM-DD)")
	cashflowTrendsCmd.Flags().StringVar(&cfStartDate, "from", "", "start date (YYYY-MM-DD)")
	cashflowTrendsCmd.Flags().StringVar(&cfEndDate, "to", "", "end date (YYYY-MM-DD)")
	cashflowTrendsCmd.Flags().StringVar(&cfTrendGroupBy, "group-by", "category", "group dimension: category or category-group")
	cashflowTrendsCmd.Flags().StringVar(&cfTrendPeriod, "period", "month", "period bucket: month, quarter, or year")
	cashflowTrendsCmd.Flags().StringSliceVar(&cfTrendAccountIDs, "account-id", nil, "account id filter (repeatable)")
	cashflowTrendsCmd.Flags().StringSliceVar(&cfTrendCategoryIDs, "category-id", nil, "category id filter (repeatable)")

	cashflowCmd.AddCommand(cashflowListCmd)
	cashflowCmd.AddCommand(cashflowSummaryCmd)
	cashflowCmd.AddCommand(cashflowCategoriesCmd)
	cashflowCmd.AddCommand(cashflowMerchantsCmd)
	cashflowCmd.AddCommand(cashflowTrendsCmd)
	cashflowCmd.AddCommand(cashflowSpendingCmd)
	RootCmd.AddCommand(cashflowCmd)
}

var cashflowSpendingCmd = &cobra.Command{
	Use:   "spending",
	Short: "Get spending breakdown by category with totals",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "cashflow.spending", start)
		if !ok {
			return
		}
		svc := deps.Service

		setCashflowDates()

		records, err := svc.GetCashflowCategories(cmd.Context(), cfStartDate, cfEndDate)
		if err != nil {
			handleError(renderer, "cashflow.spending", wrapError(err, "failed to get spending data"), start)
			return
		}

		var totalIncome, totalExpenses float64
		for _, r := range records {
			if r.Amount > 0 {
				totalIncome += r.Amount
			} else {
				totalExpenses += -r.Amount
			}
		}

		if jsonMode {
			data := map[string]interface{}{
				"period":         map[string]string{"start_date": cfStartDate, "end_date": cfEndDate},
				"total_income":   totalIncome,
				"total_expenses": totalExpenses,
				"net":            totalIncome - totalExpenses,
				"by_category":    records,
			}
			env := output.NewEnvelope("cashflow.spending", profile, output.SchemaVersion, "", data, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Spending Summary (%s to %s):\n\n", cfStartDate, cfEndDate)
			fmt.Printf("%-30s %10s\n", "CATEGORY", "AMOUNT")
			for _, r := range records {
				fmt.Printf("%-30s %10.2f\n", r.Name, r.Amount)
			}
			fmt.Printf("\n%-30s %10.2f\n", "Total Income:", totalIncome)
			fmt.Printf("%-30s %10.2f\n", "Total Expenses:", totalExpenses)
			fmt.Printf("%-30s %10.2f\n", "Net:", totalIncome-totalExpenses)
		}
	},
}
