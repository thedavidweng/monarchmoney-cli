package cli

import (
	"fmt"
	"time"

	"github.com/monarchmoney-cli/monarch/internal/auth"
	"github.com/monarchmoney-cli/monarch/internal/config"
	"github.com/monarchmoney-cli/monarch/internal/errors"
	"github.com/monarchmoney-cli/monarch/internal/graphql"
	"github.com/monarchmoney-cli/monarch/internal/monarch"
	"github.com/monarchmoney-cli/monarch/internal/output"
	"github.com/spf13/cobra"
)

var (
	startDate string
	endDate   string
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

		if startDate == "" {
			// Default to current month
			now := time.Now()
			startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		}
		if endDate == "" {
			now := time.Now()
			endDate = now.Format("2006-01-02")
		}

		summary, err := svc.GetCashflowSummary(cmd.Context(), startDate, endDate)
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
			env := output.NewEnvelope("cashflow.summary", profile, "2026-05-08", "", summary, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Cashflow Summary (%s to %s):\n", startDate, endDate)
			fmt.Printf("Income:       %.2f\n", summary.Income)
			fmt.Printf("Expense:      %.2f\n", summary.Expense)
			fmt.Printf("Savings:      %.2f\n", summary.Savings)
			fmt.Printf("Savings Rate: %.2f%%\n", summary.SavingsRate*100)
		}
	},
}

func init() {
	cashflowSummaryCmd.Flags().StringVar(&startDate, "from", "", "start date (YYYY-MM-DD)")
	cashflowSummaryCmd.Flags().StringVar(&endDate, "to", "", "end date (YYYY-MM-DD)")

	cashflowCmd.AddCommand(cashflowSummaryCmd)
	RootCmd.AddCommand(cashflowCmd)
}
