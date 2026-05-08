package cli

import (
	"fmt"
	"time"

	"github.com/monarchmoney-cli/monarch/internal/audit"
	"github.com/monarchmoney-cli/monarch/internal/auth"
	"github.com/monarchmoney-cli/monarch/internal/config"
	"github.com/monarchmoney-cli/monarch/internal/errors"
	"github.com/monarchmoney-cli/monarch/internal/graphql"
	"github.com/monarchmoney-cli/monarch/internal/monarch"
	"github.com/monarchmoney-cli/monarch/internal/output"
	"github.com/monarchmoney-cli/monarch/internal/safety"
	"github.com/spf13/cobra"
)

var (
	recurringAmount float64
)

var recurringCmd = &cobra.Command{
	Use:   "recurring",
	Short: "Manage recurring transactions",
}

var recurringListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recurring transactions",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "recurring.list", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		recurring, err := svc.ListRecurring(cmd.Context())
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list recurring transactions", errors.CatAPI, false, err)
			}
			handleError(renderer, "recurring.list", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("recurring.list", profile, "2026-05-08", "", recurring, time.Since(start))
			renderer.RenderSuccess(env)
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
		logger := audit.NewLogger()
		id := args[0]

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "recurring.update", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("recurring.update", id, nil, map[string]interface{}{"amount": recurringAmount})
			env := output.NewEnvelope("recurring.update", profile, "2026-05-08", "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "recurring.update", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		r, err := svc.UpdateRecurring(cmd.Context(), id, recurringAmount)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{
			Command:    "recurring.update",
			ResourceID: id,
			DryRun:     dryRun,
			Confirmed:  confirm,
			Profile:    profile,
			Result:     result,
			ErrorCode:  errCode,
		})

		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to update recurring transaction", errors.CatAPI, false, err)
			}
			handleError(renderer, "recurring.update", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("recurring.update", profile, "2026-05-08", "", r, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully updated recurring transaction %s.\n", r.ID)
		}
	},
}

func init() {
	recurringUpdateCmd.Flags().Float64Var(&recurringAmount, "amount", 0, "new recurring amount")
	recurringUpdateCmd.MarkFlagRequired("amount")

	recurringCmd.AddCommand(recurringListCmd)
	recurringCmd.AddCommand(recurringUpdateCmd)
	RootCmd.AddCommand(recurringCmd)
}
