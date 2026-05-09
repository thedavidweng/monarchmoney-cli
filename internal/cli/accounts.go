package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/audit"
	"github.com/thedavidweng/monarchmoney-cli/internal/auth"
	"github.com/thedavidweng/monarchmoney-cli/internal/config"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
)

var (
	accountName    string
	accountBalance float64
	accountType    string
	historyFrom    string
	historyTo      string
	refreshWait    bool
	timeframe      string
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
			env := output.NewEnvelope("accounts.list", profile, output.SchemaVersion, "", accounts, time.Since(start))
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
			env := output.NewEnvelope("accounts.holdings", profile, output.SchemaVersion, "", holdings, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-20s %12s %12s %12s\n", "ID", "QUANTITY", "BASIS", "TOTAL VALUE")
			for _, h := range holdings {
				fmt.Printf("%-20s %12.2f %12.2f %12.2f\n", h.ID, h.Quantity, h.Basis, h.TotalValue)
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

		history, err := svc.GetAccountHistory(cmd.Context(), args[0], historyFrom, historyTo)
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
			env := envelopeWithWarnings("accounts.history", history, start, "uses aggregateSnapshots for account history; per-account snapshots are not currently available")
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
	Use:   "refresh [account-id...]",
	Short: "Request a refresh of all accounts (or specific ones)",
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
			plan.Add("accounts.refresh", "", nil, map[string]interface{}{"account_ids": args})
			env := output.NewEnvelope("accounts.refresh", profile, output.SchemaVersion, "", plan, time.Since(start))
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

		err = svc.RefreshAccounts(cmd.Context(), args)
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

		if refreshWait {
			renderer.PrintDiagnostic("Waiting for refresh to complete...")
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-cmd.Context().Done():
					handleError(renderer, "accounts.refresh", errors.New(errors.InternalError, "context cancelled", errors.CatInternal, false, cmd.Context().Err()), start)
					return
				case <-ticker.C:
					status, err := svc.GetAccountsRefreshStatus(cmd.Context())
					if err != nil {
						renderer.PrintDiagnostic(fmt.Sprintf("Warning: failed to check refresh status: %v", err))
						continue
					}

					if events {
						env := output.NewEnvelope("accounts.refresh.progress", profile, output.SchemaVersion, "", status, time.Since(start))
						renderer.RenderSuccess(env)
					}

					if status["is_complete"].(bool) {
						goto complete
					}
				}
			}
		}

	complete:
		if jsonMode {
			var status string
			if refreshWait {
				status = "refresh complete"
			} else {
				status = "refresh requested"
			}
			env := output.NewEnvelope("accounts.refresh", profile, output.SchemaVersion, "", map[string]string{"status": status}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			if refreshWait {
				fmt.Println("Refresh complete.")
			} else {
				fmt.Println("Refresh requested successfully.")
			}
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
			env := output.NewEnvelope("accounts.update", profile, output.SchemaVersion, "", plan, time.Since(start))
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
			env := output.NewEnvelope("accounts.update", profile, output.SchemaVersion, "", acc, time.Since(start))
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
			env := output.NewEnvelope("accounts.delete", profile, output.SchemaVersion, "", plan, time.Since(start))
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
			env := output.NewEnvelope("accounts.delete", profile, output.SchemaVersion, "", map[string]string{"status": "deleted"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully deleted account %s.\n", id)
		}
	},
}

var accountsCreateManualCmd = &cobra.Command{
	Use:   "create-manual",
	Short: "Create a manual account",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		logger := audit.NewLogger()

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "accounts.create-manual", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("accounts.create-manual", "", nil, map[string]interface{}{"name": accountName, "type": accountType, "balance": accountBalance})
			env := output.NewEnvelope("accounts.create-manual", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.create-manual", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		acc, err := svc.CreateManualAccount(cmd.Context(), accountName, accountType, accountBalance)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{
			Command:   "accounts.create-manual",
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
				cliErr = errors.New(errors.APIError, "failed to create manual account", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.create-manual", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.create-manual", profile, output.SchemaVersion, "", acc, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully created manual account %s (%s).\n", acc.DisplayName, acc.ID)
		}
	},
}

var accountsUploadHistoryCmd = &cobra.Command{
	Use:   "upload-history <account-id> <file>",
	Short: "Upload balance history for an account",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		logger := audit.NewLogger()
		id := args[0]
		path := args[1]

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "accounts.upload-history", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("accounts.upload-history", id, nil, map[string]string{"file": path})
			env := output.NewEnvelope("accounts.upload-history", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		f, err := os.Open(path)
		if err != nil {
			handleError(renderer, "accounts.upload-history", errors.New(errors.InternalError, "failed to open file", errors.CatInternal, false, err), start)
			return
		}
		defer f.Close()

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.upload-history", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		err = svc.UploadAccountBalanceHistory(cmd.Context(), id, f)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{
			Command:    "accounts.upload-history",
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
				cliErr = errors.New(errors.APIError, "failed to upload history", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.upload-history", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.upload-history", profile, output.SchemaVersion, "", map[string]string{"status": "uploaded"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully uploaded history for account %s.\n", id)
		}
	},
}

var accountsShowCmd = &cobra.Command{
	Use:   "show <account-id>",
	Short: "Show detailed information for an account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.show", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		acc, err := svc.GetAccount(cmd.Context(), args[0])
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get account", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.show", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.show", profile, output.SchemaVersion, "", acc, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("ID:       %s\n", acc.ID)
			fmt.Printf("Name:     %s\n", acc.DisplayName)
			fmt.Printf("Type:     %s\n", acc.AccountType)
			fmt.Printf("Balance:  %.2f\n", acc.DisplayBalance)
			fmt.Printf("Updated:  %s\n", acc.UpdatedAt)
		}
	},
}

var accountsTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "List all available account types",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.types", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		types, err := svc.GetAccountTypes(cmd.Context())
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get account types", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.types", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.types", profile, output.SchemaVersion, "", types, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			for _, t := range types {
				fmt.Println(t)
			}
		}
	},
}

var accountsRefreshStatusCmd = &cobra.Command{
	Use:   "refresh-status",
	Short: "Check the status of the latest account refresh",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.refresh-status", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		status, err := svc.GetAccountsRefreshStatus(cmd.Context())
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get refresh status", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.refresh-status", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.refresh-status", profile, output.SchemaVersion, "", status, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Complete:   %v\n", status["is_complete"])
			fmt.Printf("Status:     %s\n", status["status"])
			fmt.Printf("Start Time: %s\n", status["start_time"])
			fmt.Printf("End Time:   %s\n", status["end_time"])
		}
	},
}

var accountsRecentBalancesCmd = &cobra.Command{
	Use:   "recent-balances",
	Short: "Get recent daily balances for all accounts",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if historyFrom == "" {
			historyFrom = time.Now().AddDate(0, 0, -31).Format("2006-01-02")
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.recent-balances", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		res, err := svc.GetAccountRecentBalances(cmd.Context(), historyFrom)
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get recent balances", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.recent-balances", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.recent-balances", profile, output.SchemaVersion, "", res, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Println("Recent daily balances fetched.")
		}
	},
}

var accountsSnapshotsCmd = &cobra.Command{
	Use:   "snapshots",
	Short: "Get net value snapshots by account type",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if historyFrom == "" {
			historyFrom = time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.snapshots", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		res, err := svc.GetSnapshotsByAccountType(cmd.Context(), historyFrom, timeframe)
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get snapshots", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.snapshots", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.snapshots", profile, output.SchemaVersion, "", res, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Println("Account type snapshots fetched.")
		}
	},
}

var accountsAggregateSnapshotsCmd = &cobra.Command{
	Use:   "aggregate-snapshots",
	Short: "Get aggregate net value snapshots",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "accounts.aggregate-snapshots", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		res, err := svc.GetAggregateSnapshots(cmd.Context(), historyFrom, historyTo, accountType)
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get aggregate snapshots", errors.CatAPI, false, err)
			}
			handleError(renderer, "accounts.aggregate-snapshots", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("accounts.aggregate-snapshots", profile, output.SchemaVersion, "", res, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Println("Aggregate snapshots fetched.")
		}
	},
}

func init() {
	accountsCreateManualCmd.Flags().StringVar(&accountName, "name", "", "account name")
	accountsCreateManualCmd.Flags().StringVar(&accountType, "type", "cash", "account type (e.g. cash, credit, investment)")
	accountsCreateManualCmd.Flags().Float64Var(&accountBalance, "balance", 0, "initial balance")
	accountsCreateManualCmd.MarkFlagRequired("name")

	accountsUpdateCmd.Flags().StringVar(&accountName, "name", "", "new account name")
	accountsUpdateCmd.Flags().Float64Var(&accountBalance, "balance", 0, "new account balance")

	accountsHistoryCmd.Flags().StringVar(&historyFrom, "from", "", "start date (YYYY-MM-DD)")
	accountsHistoryCmd.Flags().StringVar(&historyTo, "to", "", "end date (YYYY-MM-DD)")

	accountsRefreshCmd.Flags().BoolVar(&refreshWait, "wait", false, "wait for refresh to complete")

	accountsRecentBalancesCmd.Flags().StringVar(&historyFrom, "from", "", "start date (YYYY-MM-DD)")

	accountsSnapshotsCmd.Flags().StringVar(&historyFrom, "from", "", "start date (YYYY-MM-DD)")
	accountsSnapshotsCmd.Flags().StringVar(&timeframe, "timeframe", "month", "granularity (month or year)")

	accountsAggregateSnapshotsCmd.Flags().StringVar(&historyFrom, "from", "", "start date (YYYY-MM-DD)")
	accountsAggregateSnapshotsCmd.Flags().StringVar(&historyTo, "to", "", "end date (YYYY-MM-DD)")
	accountsAggregateSnapshotsCmd.Flags().StringVar(&accountType, "type", "", "filter by account type")

	accountsCmd.AddCommand(accountsListCmd)
	accountsCmd.AddCommand(accountsShowCmd)
	accountsCmd.AddCommand(accountsTypesCmd)
	accountsCmd.AddCommand(accountsHoldingsCmd)
	accountsCmd.AddCommand(accountsHistoryCmd)
	accountsCmd.AddCommand(accountsRefreshCmd)
	accountsCmd.AddCommand(accountsRefreshStatusCmd)
	accountsCmd.AddCommand(accountsUpdateCmd)
	accountsCmd.AddCommand(accountsDeleteCmd)
	accountsCmd.AddCommand(accountsCreateManualCmd)
	accountsCmd.AddCommand(accountsUploadHistoryCmd)
	accountsCmd.AddCommand(accountsRecentBalancesCmd)
	accountsCmd.AddCommand(accountsSnapshotsCmd)
	accountsCmd.AddCommand(accountsAggregateSnapshotsCmd)
	RootCmd.AddCommand(accountsCmd)
}
