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

var institutionsCmd = &cobra.Command{
	Use:   "institutions",
	Short: "Manage financial institutions",
}

var institutionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all institutions",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "institutions.list", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		insts, err := svc.ListInstitutions(cmd.Context())
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list institutions", errors.CatAPI, false, err)
			}
			handleError(renderer, "institutions.list", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("institutions.list", profile, "2026-05-08", "", insts, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-20s %-30s %s\n", "ID", "NAME", "URL")
			for _, inst := range insts {
				fmt.Printf("%-20s %-30s %s\n", inst.ID, inst.Name, inst.URL)
			}
		}
	},
}

func init() {
	institutionsCmd.AddCommand(institutionsListCmd)
	RootCmd.AddCommand(institutionsCmd)
}
