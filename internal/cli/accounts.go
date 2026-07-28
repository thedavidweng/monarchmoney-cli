package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/audit"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
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
	balanceAtDate  string
	accountIDs     []string
)

var accountsCmd = &cobra.Command{
	Use:     "accounts",
	Short:   "Manage Monarch Money accounts",
	GroupID: "core",
	Example: "  monarch accounts list --json\n  monarch accounts show <id>\n  monarch accounts refresh --confirm",
}

var accountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "accounts.list", "failed to list accounts",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.Account, error) {
				return svc.ListAccounts(ctx)
			},
			func(accounts []monarch.Account) {
				fmt.Printf("%-20s %-15s %-15s %s\n", "ID", "NAME", "TYPE", "BALANCE")
				for _, a := range accounts {
					fmt.Printf("%-20s %-15s %-15s %.2f\n", a.ID, a.DisplayName, a.AccountType, a.DisplayBalance)
				}
			})
	},
}

var accountsHoldingsCmd = &cobra.Command{
	Use:   "holdings <account-id>",
	Short: "List holdings for an account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "accounts.holdings", "failed to get holdings",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.Holding, error) {
				return svc.GetAccountHoldings(ctx, args[0])
			},
			func(holdings []monarch.Holding) {
				fmt.Printf("%-20s %12s %12s %12s\n", "ID", "QUANTITY", "BASIS", "TOTAL VALUE")
				for _, h := range holdings {
					fmt.Printf("%-20s %12.2f %12.2f %12.2f\n", h.ID, h.Quantity, h.Basis, h.TotalValue)
				}
			})
	},
}

var accountsBalanceAtCmd = &cobra.Command{
	Use:   "balance-at",
	Short: "Get account balances at a specific date",
	Long:  "Get display balances for all accounts, or selected accounts, as of a specific date.",
	Example: `  monarch accounts balance-at --date 2026-05-10
  monarch accounts balance-at --date 2026-05-10 --account-id acc_123 --account-id acc_456 --json --pretty`,
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "accounts.balance-at", "failed to get account balances",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.AccountBalanceAt, error) {
				if _, err := time.Parse("2006-01-02", balanceAtDate); err != nil {
					return nil, errors.New(errors.InvalidArguments, "date must use YYYY-MM-DD", errors.CatValidation, false, err)
				}
				return svc.GetAccountBalancesAt(ctx, balanceAtDate, accountIDs)
			},
			func(balances []monarch.AccountBalanceAt) {
				fmt.Printf("%-20s %-30s %-15s %12s\n", "ID", "NAME", "TYPE", "BALANCE")
				for _, balance := range balances {
					fmt.Printf("%-20s %-30s %-15s %12.2f\n", balance.ID, balance.DisplayName, balance.AccountType, balance.DisplayBalance)
				}
			})
	},
}

var accountsHistoryCmd = &cobra.Command{
	Use:   "history <account-id>",
	Short: "Get balance history for an account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runWarn(cmd.Context(), "accounts.history", "failed to get history",
			[]string{"uses aggregateSnapshots for account history; per-account snapshots are not currently available"},
			func(ctx context.Context, svc *monarch.Service) ([]monarch.HistoryRecord, error) {
				return svc.GetAccountHistory(ctx, args[0], historyFrom, historyTo)
			},
			func(history []monarch.HistoryRecord) {
				fmt.Printf("%-12s %10s\n", "DATE", "AMOUNT")
				for _, r := range history {
					fmt.Printf("%-12s %10.2f\n", r.Date, r.Amount)
				}
			})
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
			handleError(renderer, "accounts.refresh", err, start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("accounts.refresh", "", nil, map[string]any{"account_ids": args})
			env := output.NewEnvelope("accounts.refresh", profile, output.SchemaVersion, requestID, plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		deps, ok := newDeps(renderer, "accounts.refresh", start)
		if !ok {
			return
		}
		svc := deps.Service

		err := svc.RefreshAccounts(cmd.Context(), args)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{ //nolint:errcheck // best-effort audit
			Command:   "accounts.refresh",
			DryRun:    dryRun,
			Confirmed: confirm,
			Profile:   profile,
			Result:    result,
			ErrorCode: errCode,
		})

		if err != nil {
			handleError(renderer, "accounts.refresh", wrapError(err, "failed to refresh accounts"), start)
			return
		}

		if refreshWait {
			renderer.PrintDiagnostic("Waiting for refresh to complete...")
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-cmd.Context().Done():
					handleError(renderer, "accounts.refresh", errors.New(errors.InternalError, "context canceled", errors.CatInternal, false, cmd.Context().Err()), start)
					return
				case <-ticker.C:
					status, err := svc.GetAccountsRefreshStatus(cmd.Context())
					if err != nil {
						renderer.PrintDiagnostic(fmt.Sprintf("Warning: failed to check refresh status: %v", err))
						continue
					}

					if events {
						env := output.NewEnvelope("accounts.refresh.progress", profile, output.SchemaVersion, requestID, status, time.Since(start))
						renderer.RenderSuccess(env)
					}

					if done, _ := status["is_complete"].(bool); done {
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
			env := output.NewEnvelope("accounts.refresh", profile, output.SchemaVersion, requestID, map[string]string{"status": status}, time.Since(start))
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
		id := args[0]
		runMutation(cmd, "accounts.update", "failed to update account", safety.TierMutation, func() (mutation, *errors.Error) {
			var name *string
			if cmd.Flags().Changed("name") {
				name = &accountName
			}
			var balance *float64
			if cmd.Flags().Changed("balance") {
				balance = &accountBalance
			}
			var acc *monarch.Account
			return mutation{
				resourceID: id,
				planAfter:  map[string]any{"name": name, "balance": balance},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					a, err := svc.UpdateAccount(ctx, id, name, balance)
					if err != nil {
						return nil, err
					}
					acc = a
					return a, nil
				},
				human: func() { fmt.Printf("Successfully updated account %s.\n", acc.ID) },
			}, nil
		})
	},
}

var accountsDeleteCmd = &cobra.Command{
	Use:   "delete <account-id>",
	Short: "Delete an account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "accounts.delete", "failed to delete account", safety.TierDestructive, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.DeleteAccount(ctx, id); err != nil {
						return nil, err
					}
					return map[string]string{"status": "deleted"}, nil
				},
				human: func() { fmt.Printf("Successfully deleted account %s.\n", id) },
			}, nil
		})
	},
}

var accountsCreateManualCmd = &cobra.Command{
	Use:   "create-manual",
	Short: "Create a manual account",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "accounts.create-manual", "failed to create manual account", safety.TierMutation, func() (mutation, *errors.Error) {
			var acc *monarch.Account
			return mutation{
				planAfter: map[string]any{"name": accountName, "type": accountType, "balance": accountBalance},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					a, err := svc.CreateManualAccount(ctx, accountName, accountType, accountBalance)
					if err != nil {
						return nil, err
					}
					acc = a
					return a, nil
				},
				human: func() {
					fmt.Printf("Successfully created manual account %s (%s).\n", acc.DisplayName, acc.ID)
				},
			}, nil
		})
	},
}

var accountsUploadHistoryCmd = &cobra.Command{
	Use:   "upload-history <account-id> <file>",
	Short: "Upload balance history for an account",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		path := args[1]
		runMutation(cmd, "accounts.upload-history", "failed to upload history", safety.TierMutation, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				planAfter:  map[string]string{"file": path},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					f, err := os.Open(path)
					if err != nil {
						return nil, errors.New(errors.InternalError, "failed to open file", errors.CatInternal, false, err)
					}
					defer func() {
						if cerr := f.Close(); cerr != nil {
							fmt.Fprintf(os.Stderr, "warning: failed to close file: %v\n", cerr)
						}
					}()
					if err := svc.UploadAccountBalanceHistory(ctx, id, f); err != nil {
						return nil, err
					}
					return map[string]string{"status": "uploaded"}, nil
				},
				human: func() { fmt.Printf("Successfully uploaded history for account %s.\n", id) },
			}, nil
		})
	},
}

var accountsShowCmd = &cobra.Command{
	Use:   "show <account-id>",
	Short: "Show detailed information for an account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "accounts.show", "failed to get account",
			func(ctx context.Context, svc *monarch.Service) (*monarch.Account, error) {
				return svc.GetAccount(ctx, args[0])
			},
			func(acc *monarch.Account) {
				fmt.Printf("ID:       %s\n", acc.ID)
				fmt.Printf("Name:     %s\n", acc.DisplayName)
				fmt.Printf("Type:     %s\n", acc.AccountType)
				fmt.Printf("Balance:  %.2f\n", acc.DisplayBalance)
				fmt.Printf("Updated:  %s\n", acc.UpdatedAt)
			})
	},
}

var accountsTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "List all available account types",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "accounts.types", "failed to get account types",
			func(ctx context.Context, svc *monarch.Service) ([]string, error) {
				return svc.GetAccountTypes(ctx)
			},
			func(types []string) {
				for _, t := range types {
					fmt.Println(t)
				}
			})
	},
}

var accountsRefreshStatusCmd = &cobra.Command{
	Use:   "refresh-status",
	Short: "Check the status of the latest account refresh",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "accounts.refresh-status", "failed to get refresh status",
			func(ctx context.Context, svc *monarch.Service) (map[string]any, error) {
				return svc.GetAccountsRefreshStatus(ctx)
			},
			func(status map[string]any) {
				fmt.Printf("Complete:   %v\n", status["is_complete"])
				fmt.Printf("Status:     %s\n", status["status"])
				fmt.Printf("Start Time: %s\n", status["start_time"])
				fmt.Printf("End Time:   %s\n", status["end_time"])
			})
	},
}

var accountsRecentBalancesCmd = &cobra.Command{
	Use:   "recent-balances",
	Short: "Get recent daily balances for all accounts",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "accounts.recent-balances", "failed to get recent balances",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.AccountRecentBalance, error) {
				if historyFrom == "" {
					historyFrom = time.Now().AddDate(0, 0, -31).Format("2006-01-02")
				}
				return svc.GetAccountRecentBalances(ctx, historyFrom)
			},
			func(_ []monarch.AccountRecentBalance) {
				fmt.Println("Recent daily balances fetched.")
			})
	},
}

var accountsSnapshotsCmd = &cobra.Command{
	Use:   "snapshots",
	Short: "Get net value snapshots by account type",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "accounts.snapshots", "failed to get snapshots",
			func(ctx context.Context, svc *monarch.Service) (any, error) {
				if historyFrom == "" {
					historyFrom = time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
				}
				return svc.GetSnapshotsByAccountType(ctx, historyFrom, timeframe)
			},
			func(_ any) {
				fmt.Println("Account type snapshots fetched.")
			})
	},
}

var accountsAggregateSnapshotsCmd = &cobra.Command{
	Use:   "aggregate-snapshots",
	Short: "Get aggregate net value snapshots",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "accounts.aggregate-snapshots", "failed to get aggregate snapshots",
			func(ctx context.Context, svc *monarch.Service) (any, error) {
				return svc.GetAggregateSnapshots(ctx, historyFrom, historyTo, accountType)
			},
			func(_ any) {
				fmt.Println("Aggregate snapshots fetched.")
			})
	},
}

var networthCmd = &cobra.Command{
	Use:   "networth",
	Short: "Get net worth history over time",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "networth", "failed to get net worth data",
			func(ctx context.Context, svc *monarch.Service) (any, error) {
				return svc.GetAggregateSnapshots(ctx, historyFrom, historyTo, accountType)
			},
			func(_ any) {
				fmt.Println("Net worth snapshots fetched.")
			})
	},
}

func init() {
	accountsCreateManualCmd.Flags().StringVar(&accountName, "name", "", "account name")
	accountsCreateManualCmd.Flags().StringVar(&accountType, "type", "cash", "account type (e.g. cash, credit, investment)")
	accountsCreateManualCmd.Flags().Float64Var(&accountBalance, "balance", 0, "initial balance")
	accountsCreateManualCmd.MarkFlagRequired("name") //nolint:errcheck // flag registered above

	accountsUpdateCmd.Flags().StringVar(&accountName, "name", "", "new account name")
	accountsUpdateCmd.Flags().Float64Var(&accountBalance, "balance", 0, "new account balance")

	accountsHistoryCmd.Flags().StringVar(&historyFrom, "from", "", "start date (YYYY-MM-DD)")
	accountsHistoryCmd.Flags().StringVar(&historyTo, "to", "", "end date (YYYY-MM-DD)")

	accountsBalanceAtCmd.Flags().StringVar(&balanceAtDate, "date", "", "balance date (YYYY-MM-DD)")
	accountsBalanceAtCmd.Flags().StringSliceVar(&accountIDs, "account-id", nil, "account id to include (repeatable)")
	accountsBalanceAtCmd.MarkFlagRequired("date") //nolint:errcheck // flag registered above

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
	accountsCmd.AddCommand(accountsBalanceAtCmd)
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

	networthCmd.Flags().StringVar(&historyFrom, "from", "", "start date (YYYY-MM-DD)")
	networthCmd.Flags().StringVar(&historyTo, "to", "", "end date (YYYY-MM-DD)")
	networthCmd.Flags().StringVar(&accountType, "type", "", "filter by account type")
	RootCmd.AddCommand(networthCmd)
}
