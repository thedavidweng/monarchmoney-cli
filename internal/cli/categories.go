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

var categoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "Manage Monarch Money categories",
}

var categoriesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all categories",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "categories.list", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		cats, err := svc.ListCategories(cmd.Context())
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list categories", errors.CatAPI, false, err)
			}
			handleError(renderer, "categories.list", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("categories.list", profile, "2026-05-08", "", cats, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-20s %-30s %s\n", "ID", "NAME", "GROUP")
			for _, c := range cats {
				fmt.Printf("%-20s %-30s %s\n", c.ID, c.Name, c.GroupName)
			}
		}
	},
}

func init() {
	categoriesCmd.AddCommand(categoriesListCmd)
	RootCmd.AddCommand(categoriesCmd)
}
