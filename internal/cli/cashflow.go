package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/auth"
	"github.com/thedavidweng/monarchmoney-cli/internal/config"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

var (
	cfStartDate string
	cfEndDate   string
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

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "cashflow.summary", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		setCashflowDates()

		summary, err := svc.GetCashflowSummary(cmd.Context(), cfStartDate, cfEndDate)
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get cashflow summary", errors.CatAPI, false, err)
			}
			handleError(renderer, "cashflow.summary", cliErr, start)
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

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "cashflow.categories", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		setCashflowDates()

		records, err := svc.GetCashflowCategories(cmd.Context(), cfStartDate, cfEndDate)
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get cashflow categories", errors.CatAPI, false, err)
			}
			handleError(renderer, "cashflow.categories", cliErr, start)
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

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "cashflow.merchants", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		setCashflowDates()

		records, err := svc.GetCashflowMerchants(cmd.Context(), cfStartDate, cfEndDate)
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get cashflow merchants", errors.CatAPI, false, err)
			}
			handleError(renderer, "cashflow.merchants", cliErr, start)
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

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "cashflow.list", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		setCashflowDates()

		records, err := svc.ListCashflow(cmd.Context(), cfStartDate, cfEndDate)
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list cashflow", errors.CatAPI, false, err)
			}
			handleError(renderer, "cashflow.list", cliErr, start)
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

	cashflowCmd.AddCommand(cashflowListCmd)
	cashflowCmd.AddCommand(cashflowSummaryCmd)
	cashflowCmd.AddCommand(cashflowCategoriesCmd)
	cashflowCmd.AddCommand(cashflowMerchantsCmd)
	RootCmd.AddCommand(cashflowCmd)
}
