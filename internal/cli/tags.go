package cli

import (
	"fmt"
	"time"

	"github.com/monarchmoney-cli/monarch/internal/audit"
	"github.com/monarchmoney-cli/monarch/internal/auth"
	"github.com/monarchmoney-cli/monarch/internal/config"
	"github.com/monarchmoney-cli/monarch/internal/errors"
	"github.com/monarchmoney-cli/monarch/internal/graphql"
	"github.com/monarchmoney-cli/monarch/internal/monarch"
	"github.com/monarchmoney-cli/monarch/internal/output"
	"github.com/monarchmoney-cli/monarch/internal/safety"
	"github.com/spf13/cobra"
)

var (
	tagName  string
	tagColor string
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

var tagsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a tag",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		logger := audit.NewLogger()

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "tags.create", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("tags.create", "", nil, map[string]string{"name": tagName, "color": tagColor})
			env := output.NewEnvelope("tags.create", profile, "2026-05-08", "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "tags.create", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		tag, err := svc.CreateTag(cmd.Context(), tagName, tagColor)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{
			Command:   "tags.create",
			DryRun:    dryRun,
			Confirmed: confirm,
			Profile:   profile,
			Result:    result,
			ErrorCode: errCode,
		})

		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to create tag", errors.CatAPI, false, err)
			}
			handleError(renderer, "tags.create", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("tags.create", profile, "2026-05-08", "", tag, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully created tag %s (%s).\n", tag.Name, tag.ID)
		}
	},
}

func init() {
	tagsCreateCmd.Flags().StringVar(&tagName, "name", "", "tag name")
	tagsCreateCmd.Flags().StringVar(&tagColor, "color", "#000000", "tag color")
	tagsCreateCmd.MarkFlagRequired("name")

	tagsCmd.AddCommand(tagsListCmd)
	tagsCmd.AddCommand(tagsCreateCmd)
	RootCmd.AddCommand(tagsCmd)
}
