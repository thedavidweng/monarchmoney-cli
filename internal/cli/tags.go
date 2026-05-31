package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/audit"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
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

		deps, ok := newDeps(renderer, "tags.list", start)
		if !ok {
			return
		}
		svc := deps.Service

		tags, err := svc.ListTags(cmd.Context())
		if err != nil {
			handleError(renderer, "tags.list", wrapError(err, "failed to list tags"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("tags.list", profile, output.SchemaVersion, "", tags, time.Since(start))
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
			env := output.NewEnvelope("tags.create", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		deps, ok := newDeps(renderer, "tags.create", start)
		if !ok {
			return
		}
		svc := deps.Service

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
			handleError(renderer, "tags.create", wrapError(err, "failed to create tag"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("tags.create", profile, output.SchemaVersion, "", tag, time.Since(start))
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
