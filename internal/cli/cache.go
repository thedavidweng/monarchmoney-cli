package cli

import (
	stderrors "errors"
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

func openCache(renderer *output.Renderer, command string, start time.Time) (*cache.Store, bool) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		handleError(renderer, command, errors.New(errors.InternalError, "failed to load config", errors.CatInternal, false, err), start)
		return nil, false
	}
	store, err := cache.NewStore(cfg.CachePath)
	if err != nil {
		msg := "failed to open cache"
		if stderrors.Is(err, cache.ErrSchemaOutdated) {
			msg = "cache schema is outdated; run 'monarch cache sync' to rebuild it"
		}
		handleError(renderer, command, errors.New(errors.InternalError, msg, errors.CatInternal, false, err), start)
		return nil, false
	}
	return store, true
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
	Long: `Pull a full-fidelity archive copy of your Monarch data into the local cache:
accounts with type groups and lifecycle flags, transactions with tags, splits,
review state, category groups and raw merchant names, plus investment holdings
and closing balances. A cache created by an older version is rebuilt automatically.`,
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

		cfg, _ := config.Load(cfgFile)
		cacheStore, err := cache.NewStore(cfg.CachePath)
		if stderrors.Is(err, cache.ErrSchemaOutdated) {
			renderer.PrintDiagnostic("Cache schema outdated; rebuilding...")
			cacheStore, err = cache.RebuildStore(cfg.CachePath)
		}
		if err != nil {
			handleError(renderer, "cache.sync", errors.New(errors.InternalError, "failed to open cache", errors.CatInternal, false, err), start)
			return
		}
		defer cacheStore.Close()

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
				TypeGroup:      a.TypeGroup,
				DisplayBalance: a.DisplayBalance,
				CurrentBalance: a.CurrentBalance,
				IsManual:       a.IsManual,
				IsHidden:       a.IsHidden,
				IsClosed:       a.IsClosed,
				UpdatedAt:      updatedAt,
			})
		}
		if err := cacheStore.SaveAccounts(cacheAccs); err != nil {
			handleError(renderer, "cache.sync", errors.New(errors.InternalError, "failed to save accounts to cache", errors.CatInternal, false, err), start)
			return
		}

		renderer.PrintDiagnostic("Syncing transactions...")
		limit := syncLimit
		if limit <= 0 {
			limit = 1000
		}
		var txs []monarch.Transaction
		if syncAll {
			txs, err = svc.ListAllTransactions(cmd.Context(), &monarch.ListTransactionsOptions{Limit: limit, StartDate: syncFrom})
		} else {
			txs, _, err = svc.ListTransactions(cmd.Context(), &monarch.ListTransactionsOptions{Limit: limit, StartDate: syncFrom})
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
			ct := cache.Transaction{
				ID:                  t.ID,
				Date:                date,
				Amount:              t.Amount,
				Merchant:            t.Merchant,
				PlaidName:           t.PlaidName,
				ProviderDescription: t.DataProviderDescription,
				Category:            t.Category,
				CategoryGroup:       t.CategoryGroup.Name,
				CategoryGroupType:   t.CategoryGroup.Type,
				Notes:               t.Notes,
				Pending:             t.Pending,
				HideFromReports:     t.HideFromReports,
				IsRecurring:         t.IsRecurring,
				ReviewStatus:        t.ReviewStatus,
				NeedsReview:         t.NeedsReview,
				GoalID:              t.Goal.ID,
				GoalName:            t.Goal.Name,
				AccountID:           t.AccountID,
			}
			for _, tag := range t.Tags {
				ct.Tags = append(ct.Tags, cache.Tag{ID: tag.ID, Name: tag.Name})
			}
			for _, split := range t.Splits {
				ct.Splits = append(ct.Splits, cache.Split{ID: split.ID, Amount: split.Amount, Category: split.Category, Merchant: split.Merchant, Notes: split.Notes})
			}
			cacheTxs = append(cacheTxs, ct)
		}
		if len(cacheTxs) > 0 {
			if err := cacheStore.SaveTransactions(cacheTxs); err != nil {
				handleError(renderer, "cache.sync", errors.New(errors.InternalError, "failed to save transactions to cache", errors.CatInternal, false, err), start)
				return
			}
		}

		renderer.PrintDiagnostic("Syncing holdings...")
		holdings, err := svc.ListHoldings(cmd.Context())
		if err != nil {
			handleError(renderer, "cache.sync", errors.New(errors.APIError, fmt.Sprintf("failed to sync holdings: %v", err), errors.CatAPI, false, err), start)
			return
		}
		cacheHoldings := make([]cache.Holding, len(holdings))
		for i := range holdings {
			h := &holdings[i]
			cacheHoldings[i] = cache.Holding{ID: h.ID, Ticker: h.Ticker, Name: h.Name, Quantity: h.Quantity, Basis: h.Basis, Value: h.Value, AccountID: h.AccountID}
		}
		if err := cacheStore.SaveHoldings(cacheHoldings); err != nil {
			handleError(renderer, "cache.sync", errors.New(errors.InternalError, "failed to save holdings to cache", errors.CatInternal, false, err), start)
			return
		}

		if err := cacheStore.RecordSync(len(cacheAccs), len(cacheTxs)); err != nil {
			handleError(renderer, "cache.sync", errors.New(errors.InternalError, "failed to record sync metadata", errors.CatInternal, false, err), start)
			return
		}

		var backupPath string
		var backupWarnings []string
		if cfg.BackupPath != "" {
			renderer.PrintDiagnostic("Regenerating ledger backup...")
			if _, warn, err := writeJournal(cacheStore, cfg.BackupPath); err != nil {
				backupWarnings = append(backupWarnings, fmt.Sprintf("ledger backup regeneration failed: %v", err))
				renderer.PrintDiagnostic(fmt.Sprintf("Ledger backup regeneration failed: %v", err))
			} else {
				backupWarnings = append(backupWarnings, warn...)
				backupPath = cfg.BackupPath
			}
		}

		if jsonMode {
			data := map[string]any{"status": "sync complete", "accounts": len(cacheAccs), "transactions": len(cacheTxs), "holdings": len(cacheHoldings)}
			if backupPath != "" {
				data["backup"] = backupPath
			}
			env := output.NewEnvelope("cache.sync", profile, output.SchemaVersion, requestID, data, time.Since(start))
			env.Meta.Warnings = backupWarnings
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Sync complete. %d accounts, %d transactions, %d holdings.\n", len(cacheAccs), len(cacheTxs), len(cacheHoldings))
			if backupPath != "" {
				fmt.Printf("Ledger backup written to %s.\n", backupPath)
			}
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

		cacheStore, ok := openCache(renderer, "cache.search", start)
		if !ok {
			return
		}
		defer cacheStore.Close()

		txs, err := cacheStore.SearchTransactions(args[0])
		if err != nil {
			handleError(renderer, "cache.search", errors.New(errors.InternalError, "search failed", errors.CatInternal, false, err), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("cache.search", profile, output.SchemaVersion, requestID, txs, time.Since(start))
			renderer.RenderSuccess(env)
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

		cacheStore, ok := openCache(renderer, "cache.stats", start)
		if !ok {
			return
		}
		defer cacheStore.Close()

		stats, _ := cacheStore.GetStats()

		if jsonMode {
			env := output.NewEnvelope("cache.stats", profile, output.SchemaVersion, requestID, stats, time.Since(start))
			renderer.RenderSuccess(env)
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

		store, ok := openCache(renderer, "cache.cleanup", start)
		if !ok {
			return
		}
		defer store.Close()

		affected, err := store.Cleanup(cleanupBefore)
		if err != nil {
			handleError(renderer, "cache.cleanup", errors.New(errors.InternalError, "failed to cleanup cache", errors.CatInternal, false, err), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("cache.cleanup", profile, output.SchemaVersion, requestID, map[string]int64{"deleted": affected}, time.Since(start))
			renderer.RenderSuccess(env)
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
