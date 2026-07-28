package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
)

var (
	overviewFrom string
	overviewTo   string
)

var overviewCmd = &cobra.Command{
	Use:     "overview",
	Short:   "Get a compact financial overview",
	GroupID: "core",
	Example: "  monarch overview\n  monarch overview --from 2026-01-01 --to 2026-01-31 --json",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "overview", "failed to get financial overview",
			func(ctx context.Context, svc *monarch.Service) (*monarch.FinancialOverview, error) {
				return svc.GetFinancialOverview(ctx, overviewFrom, overviewTo)
			},
			func(ov *monarch.FinancialOverview) {
				fmt.Printf("Financial Overview (as of %s)\n\n", ov.AsOf)
				fmt.Printf("Net Worth:       %.2f\n", ov.NetWorth)
				fmt.Printf("Accounts:        %d\n", ov.AccountCount)
				if ov.Cashflow != nil {
					fmt.Printf("Income:          %.2f\n", ov.Cashflow.Income)
					fmt.Printf("Expense:         %.2f\n", ov.Cashflow.Expense)
					fmt.Printf("Savings:         %.2f\n", ov.Cashflow.Savings)
					fmt.Printf("Savings Rate:    %.2f%%\n", ov.Cashflow.SavingsRate*100)
				}
				fmt.Printf("Transactions:    %d total (showing %d)\n\n", ov.TransactionTotal, len(ov.Transactions))
				if len(ov.Transactions) > 0 {
					fmt.Printf("%-12s %-20s %-15s %10s\n", "DATE", "MERCHANT", "CATEGORY", "AMOUNT")
					for _, tx := range ov.Transactions {
						merchant := tx.Merchant
						category := tx.Category
						fmt.Printf("%-12s %-20s %-15s %10.2f\n", tx.Date, merchant, category, tx.Amount)
					}
				}
			})
	},
}

func init() {
	overviewCmd.Flags().StringVar(&overviewFrom, "from", "", "start date (YYYY-MM-DD, defaults to current month)")
	overviewCmd.Flags().StringVar(&overviewTo, "to", "", "end date (YYYY-MM-DD, defaults to current month)")
	RootCmd.AddCommand(overviewCmd)
}
