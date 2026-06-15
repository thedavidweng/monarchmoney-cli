package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
)

var (
	recurringAmount float64
)

var recurringCmd = &cobra.Command{
	Use:     "recurring",
	Short:   "Manage recurring transactions",
	GroupID: "core",
	Example: "  monarch recurring list --json",
}

var recurringListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recurring transactions",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "recurring.list", start)
		if !ok {
			return
		}
		svc := deps.Service

		// Default to current month
		now := time.Now()
		startDate := now.Format("2006-01-02")
		endDate := time.Date(now.Year(), now.Month()+2, 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02")

		recurring, err := svc.ListRecurring(cmd.Context(), startDate, endDate)
		if err != nil {
			handleError(renderer, "recurring.list", wrapError(err, "failed to list recurring transactions"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("recurring.list", profile, output.SchemaVersion, "", recurring, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("%-20s %10s %-12s %-12s %s\n", "MERCHANT", "AMOUNT", "FREQUENCY", "NEXT DATE", "STATUS")
			for _, r := range recurring {
				fmt.Printf("%-20s %10.2f %-12s %-12s %s\n", r.Merchant, r.Amount, r.Frequency, r.NextDate, r.Status)
			}
		}
	},
}

var recurringUpdateCmd = &cobra.Command{
	Use:   "update <recurring-id>",
	Short: "Update a recurring transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		id := args[0]

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "recurring.update", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("recurring.update", id, nil, map[string]any{"amount": recurringAmount})
			env := output.NewEnvelope("recurring.update", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
			return
		}

		deps, ok := newDeps(renderer, "recurring.update", start)
		if !ok {
			return
		}

		result, err := deps.Mutate("recurring.update", id, func() (any, error) {
			return deps.Service.UpdateRecurring(cmd.Context(), id, recurringAmount)
		}, "failed to update recurring transaction")
		if err != nil {
			return
		}
		r := result.(*monarch.RecurringTransaction)

		if jsonMode {
			env := output.NewEnvelope("recurring.update", profile, output.SchemaVersion, "", r, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("Successfully updated recurring transaction %s.\n", r.ID)
		}
	},
}

func init() {
	recurringUpdateCmd.Flags().Float64Var(&recurringAmount, "amount", 0, "new recurring amount")
	recurringUpdateCmd.MarkFlagRequired("amount") //nolint:errcheck // flag registered above

	recurringCmd.AddCommand(recurringListCmd)
	recurringCmd.AddCommand(recurringUpdateCmd)
	RootCmd.AddCommand(recurringCmd)
}
