package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/cache"
	"github.com/thedavidweng/monarchmoney-cli/internal/config"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

func parseCacheDate(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	return time.Parse("2006-01-02", value)
}

var cacheCmd = &cobra.Command{
	Use:     "cache",
	Short:   "Manage local data cache",
	GroupID: "utility",
	Example: "  monarch cache sync\n  monarch cache search \"grocery\"\n  monarch cache stats",
}

var cacheSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync data from Monarch to local cache",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if syncFrom != "" {
			if _, err := time.Parse("2006-01-02", syncFrom); err != nil {
				handleError(renderer, "cache.sync", errors.New(errors.InvalidArguments, "--from must be a date in YYYY-MM-DD format", errors.CatValidation, false, err), start)
				return
			}
		}

		deps, ok := newDeps(renderer, "cache.sync", start)
		if !ok {
			return
		}
		svc := deps.Service

		cfg, _ := config.Load()
		cacheStore, err := cache.NewStore(cfg.CachePath)
		if err != nil {
			handleError(renderer, "cache.sync", errors.New(errors.InternalError, "failed to open cache", errors.CatInternal, false, err), start)
			return
		}
		defer cacheStore.Close() //nolint:errcheck // best-effort close

		// Sync accounts
		renderer.PrintDiagnostic("Syncing accounts...")
		accounts, err := svc.ListAccounts(cmd.Context())
		if err != nil {
			handleError(renderer, "cache.sync", errors.New(errors.APIError, fmt.Sprintf("failed to sync accounts: %v", err), errors.CatAPI, false, err), start)
			return
		}
		var cacheAccs []cache.Account
		for _, a := range accounts {
			updatedAt, err := parseCacheDate(a.UpdatedAt)
			if err != nil {
				handleError(renderer, "cache.sync", errors.New(errors.APISchemaChanged, "failed to parse account updated_at", errors.CatAPI, false, err), start)
				return
			}
			cacheAccs = append(cacheAccs, cache.Account{
				ID:             a.ID,
				DisplayName:    a.DisplayName,
				AccountType:    a.AccountType,
				DisplayBalance: a.DisplayBalance,
				UpdatedAt:      updatedAt,
			})
		}
		if err := cacheStore.SaveAccounts(cacheAccs); err != nil {
			handleError(renderer, "cache.sync", errors.New(errors.InternalError, "failed to save accounts to cache", errors.CatInternal, false, err), start)
			return
		}

		// Sync transactions with pagination when --all is set.
		renderer.PrintDiagnostic("Syncing transactions...")
		limit := syncLimit
		if limit <= 0 {
			limit = 1000
		}
		var txs []monarch.Transaction
		if syncAll {
			txs, err = svc.ListAllTransactions(cmd.Context(), monarch.ListTransactionsOptions{Limit: limit, StartDate: syncFrom})
		} else {
			txs, _, err = svc.ListTransactions(cmd.Context(), monarch.ListTransactionsOptions{Limit: limit, StartDate: syncFrom})
		}
		if err != nil {
			handleError(renderer, "cache.sync", errors.New(errors.APIError, fmt.Sprintf("failed to sync transactions: %v", err), errors.CatAPI, false, err), start)
			return
		}
		var cacheTxs []cache.Transaction
		for _, t := range txs {
			date, err := time.Parse("2006-01-02", t.Date)
			if err != nil {
				handleError(renderer, "cache.sync", errors.New(errors.APISchemaChanged, "failed to parse transaction date", errors.CatAPI, false, err), start)
				return
			}
			cacheTxs = append(cacheTxs, cache.Transaction{
				ID:        t.ID,
				Date:      date,
				Amount:    t.Amount,
				Merchant:  t.Merchant,
				Category:  t.Category,
				Notes:     t.Notes,
				AccountID: t.AccountID,
			})
		}
		if err := cacheStore.SaveTransactions(cacheTxs); err != nil {
			handleError(renderer, "cache.sync", errors.New(errors.InternalError, "failed to save transactions to cache", errors.CatInternal, false, err), start)
			return
		}

		cacheStore.RecordSync(len(cacheAccs), len(cacheTxs)) //nolint:errcheck // best-effort sync record

		if jsonMode {
			env := output.NewEnvelope("cache.sync", profile, output.SchemaVersion, "", map[string]any{"status": "sync complete", "accounts": len(cacheAccs), "transactions": len(cacheTxs)}, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("Sync complete. %d accounts, %d transactions.\n", len(cacheAccs), len(cacheTxs))
		}
	},
}

var cacheSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search transactions in local cache",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		cfg, _ := config.Load()
		cacheStore, err := cache.NewStore(cfg.CachePath)
		if err != nil {
			handleError(renderer, "cache.search", errors.New(errors.InternalError, "failed to open cache", errors.CatInternal, false, err), start)
			return
		}
		defer cacheStore.Close() //nolint:errcheck // best-effort close

		txs, err := cacheStore.SearchTransactions(args[0])
		if err != nil {
			handleError(renderer, "cache.search", errors.New(errors.InternalError, "search failed", errors.CatInternal, false, err), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("cache.search", profile, output.SchemaVersion, "", txs, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("%-12s %-20s %-15s %10s %s\n", "DATE", "MERCHANT", "CATEGORY", "AMOUNT", "NOTES")
			for _, t := range txs {
				fmt.Printf("%-12s %-20s %-15s %10.2f %s\n", t.Date.Format("2006-01-02"), t.Merchant, t.Category, t.Amount, t.Notes)
			}
			fmt.Printf("\nTotal matches: %d\n", len(txs))
		}
	},
}

var cacheStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show cache statistics",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		cfg, _ := config.Load()
		cacheStore, err := cache.NewStore(cfg.CachePath)
		if err != nil {
			handleError(renderer, "cache.stats", errors.New(errors.InternalError, "failed to open cache", errors.CatInternal, false, err), start)
			return
		}
		defer cacheStore.Close() //nolint:errcheck // best-effort close

		stats, _ := cacheStore.GetStats()

		if jsonMode {
			env := output.NewEnvelope("cache.stats", profile, output.SchemaVersion, "", stats, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Println("Cache Statistics")
			for k, v := range stats {
				switch val := v.(type) {
				case int64:
					fmt.Printf("%s: %d\n", k, val)
				case string:
					fmt.Printf("%s: %s\n", k, val)
				default:
					fmt.Printf("%s: %v\n", k, val)
				}
			}
		}
	},
}

var syncFrom string
var syncLimit int
var syncAll bool
var cleanupBefore string

var cacheCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up old transactions from cache",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if cleanupBefore == "" {
			handleError(renderer, "cache.cleanup", errors.New(errors.InvalidArguments, "--before is required", errors.CatValidation, false, nil), start)
			return
		}
		if _, err := time.Parse("2006-01-02", cleanupBefore); err != nil {
			handleError(renderer, "cache.cleanup", errors.New(errors.InvalidArguments, "--before must be a date in YYYY-MM-DD format", errors.CatValidation, false, err), start)
			return
		}

		cfg, _ := config.Load()
		store, err := cache.NewStore(cfg.CachePath)
		if err != nil {
			handleError(renderer, "cache.cleanup", errors.New(errors.InternalError, "failed to open cache", errors.CatInternal, false, err), start)
			return
		}
		defer store.Close() //nolint:errcheck // best-effort close

		affected, err := store.Cleanup(cleanupBefore)
		if err != nil {
			handleError(renderer, "cache.cleanup", errors.New(errors.InternalError, "failed to cleanup cache", errors.CatInternal, false, err), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("cache.cleanup", profile, output.SchemaVersion, "", map[string]int64{"deleted": affected}, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("Deleted %d transactions from cache.\n", affected)
		}
	},
}

func init() {
	cacheSyncCmd.Flags().StringVar(&syncFrom, "from", "", "sync transactions from date (YYYY-MM-DD)")
	cacheSyncCmd.Flags().IntVar(&syncLimit, "limit", 1000, "max transactions per page (default 1000)")
	cacheSyncCmd.Flags().BoolVar(&syncAll, "all", false, "paginate through all matching transactions")

	cacheCleanupCmd.Flags().StringVar(&cleanupBefore, "before", "", "delete transactions before date (YYYY-MM-DD)")
	cacheCleanupCmd.MarkFlagRequired("before") //nolint:errcheck // flag registered above

	cacheCmd.AddCommand(cacheSyncCmd)
	cacheCmd.AddCommand(cacheSearchCmd)
	cacheCmd.AddCommand(cacheStatsCmd)
	cacheCmd.AddCommand(cacheCleanupCmd)
	RootCmd.AddCommand(cacheCmd)
}
