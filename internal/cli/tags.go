package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
)

var (
	tagName   string
	tagColor  string
	tagSearch string
	tagLimit  int
	tagOrder  int
)

var tagsCmd = &cobra.Command{
	Use:     "tags",
	Short:   "Manage Monarch Money tags",
	GroupID: "core",
	Example: "  monarch tags list --json\n  monarch tags create --name \"reimbursable\" --confirm",
}

var tagsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "tags.list", "failed to list tags",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.Tag, error) {
				return svc.ListTags(ctx, tagSearch, tagLimit)
			},
			func(tags []monarch.Tag) {
				fmt.Printf("%-20s %-20s %-10s %s\n", "ID", "NAME", "ORDER", "COLOR")
				for _, t := range tags {
					fmt.Printf("%-20s %-20s %-10d %s\n", t.ID, t.Name, t.Order, t.Color)
				}
			})
	},
}

var tagsShowCmd = &cobra.Command{
	Use:   "show <tag-id>",
	Short: "Show a tag",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "tags.show", "failed to get tag",
			func(ctx context.Context, svc *monarch.Service) (*monarch.Tag, error) {
				return svc.GetTag(ctx, args[0])
			},
			func(t *monarch.Tag) {
				fmt.Printf("ID:    %s\n", t.ID)
				fmt.Printf("Name:  %s\n", t.Name)
				fmt.Printf("Color: %s\n", t.Color)
				fmt.Printf("Order: %d\n", t.Order)
			})
	},
}

var tagsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a tag (requires --confirm)",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "tags.create", "failed to create tag", safety.TierMutation, func() (mutation, *errors.Error) {
			var tag *monarch.Tag
			return mutation{
				planAfter: map[string]string{"name": tagName, "color": tagColor},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					t, err := svc.CreateTag(ctx, tagName, tagColor)
					if err != nil {
						return nil, err
					}
					tag = t
					return t, nil
				},
				human: func() { fmt.Printf("Successfully created tag %s (%s).\n", tag.Name, tag.ID) },
			}, nil
		})
	},
}

var tagsUpdateCmd = &cobra.Command{
	Use:   "update <tag-id>",
	Short: "Update a tag name or color (requires --confirm)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "tags.update", "failed to update tag", safety.TierMutation, func() (mutation, *errors.Error) {
			var name, color *string
			after := map[string]any{}
			if cmd.Flags().Changed("name") {
				name = &tagName
				after["name"] = tagName
			}
			if cmd.Flags().Changed("color") {
				color = &tagColor
				after["color"] = tagColor
			}
			if len(after) == 0 {
				return mutation{}, errors.New(errors.InvalidArguments, "at least one of --name, --color is required", errors.CatValidation, false, nil)
			}
			var tag *monarch.Tag
			return mutation{
				resourceID: id,
				planAfter:  after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateTag(ctx, id, name, color)
					if err != nil {
						return nil, err
					}
					tag = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Successfully updated tag %s (%s).\n", tag.Name, tag.ID) },
			}, nil
		})
	},
}

var tagsDeleteCmd = &cobra.Command{
	Use:   "delete <tag-id>",
	Short: "Delete a tag (requires --confirm)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "tags.delete", "failed to delete tag", safety.TierDestructive, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.DeleteTag(ctx, id); err != nil {
						return nil, err
					}
					return map[string]string{"status": "deleted"}, nil
				},
				human: func() { fmt.Printf("Successfully deleted tag %s.\n", id) },
			}, nil
		})
	},
}

var tagsReorderCmd = &cobra.Command{
	Use:   "reorder <tag-id>",
	Short: "Move a tag to a new position (requires --confirm)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "tags.reorder", "failed to reorder tag", safety.TierMutation, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				planAfter:  map[string]int{"order": tagOrder},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					tags, err := svc.ReorderTag(ctx, id, tagOrder)
					if err != nil {
						return nil, err
					}
					return tags, nil
				},
				human: func() { fmt.Printf("Successfully moved tag %s to position %d.\n", id, tagOrder) },
			}, nil
		})
	},
}

func init() {
	tagsListCmd.Flags().StringVar(&tagSearch, "search", "", "filter tags by name")
	tagsListCmd.Flags().IntVar(&tagLimit, "limit", 0, "maximum number of tags to return")

	tagsCreateCmd.Flags().StringVar(&tagName, "name", "", "tag name")
	tagsCreateCmd.Flags().StringVar(&tagColor, "color", "#000000", "tag color")
	tagsCreateCmd.MarkFlagRequired("name") //nolint:errcheck // flag registered above

	tagsUpdateCmd.Flags().StringVar(&tagName, "name", "", "new tag name")
	tagsUpdateCmd.Flags().StringVar(&tagColor, "color", "", "new tag color")

	tagsReorderCmd.Flags().IntVar(&tagOrder, "order", 0, "new position")
	tagsReorderCmd.MarkFlagRequired("order") //nolint:errcheck // flag registered above

	tagsCmd.AddCommand(tagsListCmd)
	tagsCmd.AddCommand(tagsShowCmd)
	tagsCmd.AddCommand(tagsCreateCmd)
	tagsCmd.AddCommand(tagsUpdateCmd)
	tagsCmd.AddCommand(tagsDeleteCmd)
	tagsCmd.AddCommand(tagsReorderCmd)
	RootCmd.AddCommand(tagsCmd)
}
