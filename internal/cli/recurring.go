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

func init() {
	recurringCmd.AddCommand(recurringListCmd)
	RootCmd.AddCommand(recurringCmd)
}
