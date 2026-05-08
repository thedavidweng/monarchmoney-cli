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
	accountName    string
	accountBalance float64
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage Monarch Money accounts",
}

var accountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.list", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		accounts, err := svc.ListAccounts(cmd.Context())
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list accounts", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.list", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.list", profile, "2026-05-08", "", accounts, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-20s %-15s %-15s %s\n", "ID", "NAME", "TYPE", "BALANCE")
			for _, a := range accounts {
				fmt.Printf("%-20s %-15s %-15s %.2f\n", a.ID, a.DisplayName, a.AccountType, a.DisplayBalance)
			}
		}
	},
}

var accountsHoldingsCmd = &cobra.Command{
	Use:   "holdings <account-id>",
	Short: "List holdings for an account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.holdings", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		holdings, err := svc.GetAccountHoldings(cmd.Context(), args[0])
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get holdings", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.holdings", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.holdings", profile, "2026-05-08", "", holdings, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-20s %-10s %10s %10s %10s\n", "SECURITY", "SYMBOL", "QUANTITY", "PRICE", "VALUE")
			for _, h := range holdings {
				fmt.Printf("%-20s %-10s %10.2f %10.2f %10.2f\n", h.Security, h.Symbol, h.Quantity, h.Price, h.Value)
			}
		}
	},
}

var accountsHistoryCmd = &cobra.Command{
	Use:   "history <account-id>",
	Short: "Get balance history for an account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.history", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		history, err := svc.GetAccountHistory(cmd.Context(), args[0])
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get history", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.history", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.history", profile, "2026-05-08", "", history, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-12s %10s\n", "DATE", "AMOUNT")
			for _, r := range history {
				fmt.Printf("%-12s %10.2f\n", r.Date, r.Amount)
			}
		}
	},
}

var accountsRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Request a refresh of all accounts",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		logger := audit.NewLogger()

		if err := safety.Check(safety.TierRemoteAction, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "accounts.refresh", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("accounts.refresh", "", nil, nil)
			env := output.NewEnvelope("accounts.refresh", profile, "2026-05-08", "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.refresh", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		err = svc.RefreshAccounts(cmd.Context())
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{
			Command:   "accounts.refresh",
			DryRun:    dryRun,
			Confirmed: confirm,
			Profile:   profile,
			Result:    result,
			ErrorCode: errCode,
		})

		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to refresh accounts", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.refresh", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.refresh", profile, "2026-05-08", "", map[string]string{"status": "refresh requested"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Println("Refresh requested successfully.")
		}
	},
}

var accountsUpdateCmd = &cobra.Command{
	Use:   "update <account-id>",
	Short: "Update an account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		logger := audit.NewLogger()
		id := args[0]

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "accounts.update", err.(*errors.Error), start)
			return
		}

		var name *string
		if cmd.Flags().Changed("name") {
			name = &accountName
		}
		var balance *float64
		if cmd.Flags().Changed("balance") {
			balance = &accountBalance
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("accounts.update", id, nil, map[string]interface{}{"name": name, "balance": balance})
			env := output.NewEnvelope("accounts.update", profile, "2026-05-08", "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.update", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		acc, err := svc.UpdateAccount(cmd.Context(), id, name, balance)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{
			Command:    "accounts.update",
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
				cliErr = errors.New(errors.APIError, "failed to update account", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.update", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.update", profile, "2026-05-08", "", acc, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully updated account %s.\n", acc.ID)
		}
	},
}

var accountsDeleteCmd = &cobra.Command{
	Use:   "delete <account-id>",
	Short: "Delete an account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		logger := audit.NewLogger()
		id := args[0]

		if err := safety.Check(safety.TierDestructive, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "accounts.delete", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("accounts.delete", id, nil, nil)
			env := output.NewEnvelope("accounts.delete", profile, "2026-05-08", "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.delete", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		err = svc.DeleteAccount(cmd.Context(), id)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{
			Command:    "accounts.delete",
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
				cliErr = errors.New(errors.APIError, "failed to delete account", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.delete", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.delete", profile, "2026-05-08", "", map[string]string{"status": "deleted"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully deleted account %s.\n", id)
		}
	},
}

func init() {
	accountsUpdateCmd.Flags().StringVar(&accountName, "name", "", "new account name")
	accountsUpdateCmd.Flags().Float64Var(&accountBalance, "balance", 0, "new account balance")

	accountsCmd.AddCommand(accountsListCmd)
	accountsCmd.AddCommand(accountsHoldingsCmd)
	accountsCmd.AddCommand(accountsHistoryCmd)
	accountsCmd.AddCommand(accountsRefreshCmd)
	accountsCmd.AddCommand(accountsUpdateCmd)
	accountsCmd.AddCommand(accountsDeleteCmd)
	RootCmd.AddCommand(accountsCmd)
}
