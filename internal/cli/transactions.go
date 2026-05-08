package cli

import (
	"fmt"
	"time"

	"github.com/monarchmoney-cli/monarch/internal/auth"
	"github.com/monarchmoney-cli/monarch/internal/config"
	"github.com/monarchmoney-cli/monarch/internal/errors"
	"github.com/monarchmoney-cli/monarch/internal/graphql"
	"github.com/monarchmoney-cli/monarch/internal/monarch"
	"github.com/monarchmoney-cli/monarch/internal/output"
	"github.com/spf13/cobra"
)

var (
	limit  int
	offset int
)

var transactionsCmd = &cobra.Command{
	Use:   "transactions",
	Short: "Manage Monarch Money transactions",
}

var transactionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List transactions",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "transactions.list", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		txs, total, err := svc.ListTransactions(cmd.Context(), monarch.ListTransactionsOptions{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list transactions", errors.CatAPI, false, err)
			}
			handleError(renderer, "transactions.list", cliErr, start)
			return
		}

		if jsonMode {
			data := map[string]interface{}{
				"transactions": txs,
				"total":        total,
			}
			env := output.NewEnvelope("transactions.list", profile, "2026-05-08", "", data, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-12s %-20s %-15s %10s %s\n", "DATE", "MERCHANT", "CATEGORY", "AMOUNT", "NOTES")
			for _, t := range txs {
				fmt.Printf("%-12s %-20s %-15s %10.2f %s\n", t.Date, t.Merchant, t.Category, t.Amount, t.Notes)
			}
			fmt.Printf("\nTotal transactions: %d\n", total)
		}
	},
}

func init() {
	transactionsListCmd.Flags().IntVar(&limit, "limit", 100, "maximum number of transactions to return")
	transactionsListCmd.Flags().IntVar(&offset, "offset", 0, "number of transactions to skip")

	transactionsCmd.AddCommand(transactionsListCmd)
	RootCmd.AddCommand(transactionsCmd)
}
