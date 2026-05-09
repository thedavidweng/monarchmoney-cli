package cli

import (
	"fmt"
	"time"

	"github.com/monarchmoney-cli/monarch/internal/auth"
	"github.com/monarchmoney-cli/monarch/internal/cache"
	"github.com/monarchmoney-cli/monarch/internal/config"
	"github.com/monarchmoney-cli/monarch/internal/errors"
	"github.com/monarchmoney-cli/monarch/internal/graphql"
	"github.com/monarchmoney-cli/monarch/internal/monarch"
	"github.com/monarchmoney-cli/monarch/internal/output"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage local data cache",
}

var cacheSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync data from Monarch to local cache",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "cache.sync", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		cfg, _ := config.Load()
		cacheStore, err := cache.NewStore(cfg.CachePath)
		if err != nil {
			handleError(renderer, "cache.sync", errors.New(errors.InternalError, "failed to open cache", errors.CatInternal, false, err), start)
			return
		}

		// Sync accounts
		renderer.PrintDiagnostic("Syncing accounts...")
		accounts, err := svc.ListAccounts(cmd.Context())
		if err == nil {
			var cacheAccs []cache.Account
			for _, a := range accounts {
				updatedAt, _ := time.Parse(time.RFC3339, a.UpdatedAt)
				cacheAccs = append(cacheAccs, cache.Account{
					ID:             a.ID,
					DisplayName:    a.DisplayName,
					AccountType:    a.AccountType,
					DisplayBalance: a.DisplayBalance,
					UpdatedAt:      updatedAt,
				})
			}
			cacheStore.SaveAccounts(cacheAccs)
		}

		// Sync transactions (simplified: sync last 1000)
		renderer.PrintDiagnostic("Syncing transactions...")
		txs, _, err := svc.ListTransactions(cmd.Context(), monarch.ListTransactionsOptions{Limit: 1000})
		if err == nil {
			var cacheTxs []cache.Transaction
			for _, t := range txs {
				date, _ := time.Parse("2006-01-02", t.Date)
				cacheTxs = append(cacheTxs, cache.Transaction{
					ID:       t.ID,
					Date:     date,
					Amount:   t.Amount,
					Merchant: t.Merchant,
					Category: t.Category,
					Notes:    t.Notes,
				})
			}
			cacheStore.SaveTransactions(cacheTxs)
		}

		if jsonMode {
			env := output.NewEnvelope("cache.sync", profile, "2026-05-08", "", map[string]string{"status": "sync complete"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Println("Sync complete.")
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

		txs, err := cacheStore.SearchTransactions(args[0])
		if err != nil {
			handleError(renderer, "cache.search", errors.New(errors.InternalError, "search failed", errors.CatInternal, false, err), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("cache.search", profile, "2026-05-08", "", txs, time.Since(start))
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

		cfg, _ := config.Load()
		cacheStore, err := cache.NewStore(cfg.CachePath)
		if err != nil {
			handleError(renderer, "cache.stats", errors.New(errors.InternalError, "failed to open cache", errors.CatInternal, false, err), start)
			return
		}

		stats, _ := cacheStore.GetStats()

		if jsonMode {
			env := output.NewEnvelope("cache.stats", profile, "2026-05-08", "", stats, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Println("Cache Statistics")
			for k, v := range stats {
				fmt.Printf("%s: %d\n", k, v)
			}
		}
	},
}

func init() {
	cacheCmd.AddCommand(cacheSyncCmd)
	cacheCmd.AddCommand(cacheSearchCmd)
	cacheCmd.AddCommand(cacheStatsCmd)
	RootCmd.AddCommand(cacheCmd)
}
