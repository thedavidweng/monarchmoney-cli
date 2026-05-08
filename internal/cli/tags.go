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

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Manage Monarch Money tags",
}

var tagsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "tags.list", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		tags, err := svc.ListTags(cmd.Context())
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list tags", errors.CatAPI, false, err)
			}
			handleError(renderer, "tags.list", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("tags.list", profile, "2026-05-08", "", tags, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-20s %-20s %s\n", "ID", "NAME", "COLOR")
			for _, t := range tags {
				fmt.Printf("%-20s %-20s %s\n", t.ID, t.Name, t.Color)
			}
		}
	},
}

func init() {
	tagsCmd.AddCommand(tagsListCmd)
	RootCmd.AddCommand(tagsCmd)
}
