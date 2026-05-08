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

func init() {
	accountsCmd.AddCommand(accountsListCmd)
	accountsCmd.AddCommand(accountsHoldingsCmd)
	accountsCmd.AddCommand(accountsHistoryCmd)
	RootCmd.AddCommand(accountsCmd)
}
