package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
)

var debtMethod string

var debtCmd = &cobra.Command{
	Use:     "debt",
	Short:   "Manage debt paydown plans",
	GroupID: "core",
	Example: "  monarch debt paydown --json",
}

var debtPaydownCmd = &cobra.Command{
	Use:   "paydown",
	Short: "Show the debt paydown plan and the accounts feeding it",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "debt.paydown", "failed to get debt paydown plan",
			func(ctx context.Context, svc *monarch.Service) (*monarch.DebtPaydownPlan, error) {
				return svc.GetDebtPaydown(ctx, debtMethod)
			},
			func(plan *monarch.DebtPaydownPlan) {
				fmt.Printf("Method:            %s\n", plan.Method)
				fmt.Printf("Debt in plan:      %.2f\n", plan.CurrentPrincipal)
				fmt.Printf("Projected interest: %.2f\n", plan.ProjectedInterest)
				fmt.Printf("Projected total:   %.2f\n", plan.ProjectedTotal)
				fmt.Printf("Debt-free date:    %s\n\n", plan.DebtFreeDate)
				fmt.Printf("%-30s %12s %8s %12s %12s\n", "ACCOUNT", "BALANCE", "APR", "MIN PAY", "PLANNED PAY")
				for _, a := range plan.IncludedAccounts {
					fmt.Printf("%-30s %12.2f %8s %12s %12s\n", a.Name, a.Balance, formatOptFloat(a.APR), formatOptFloat(a.MinimumPayment), formatOptFloat(a.PlannedPayment))
				}
				if len(plan.ExcludedAccounts) > 0 {
					fmt.Printf("\nExcluded from plan:\n")
					for _, a := range plan.ExcludedAccounts {
						fmt.Printf("%-30s %12.2f\n", a.Name, a.Balance)
					}
				}
			})
	},
}

func formatOptFloat(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.2f", *v)
}

func init() {
	debtPaydownCmd.Flags().StringVar(&debtMethod, "method", "planned", "paydown strategy (planned, avalanche, snowball)")
	debtCmd.AddCommand(debtPaydownCmd)
	RootCmd.AddCommand(debtCmd)
}
