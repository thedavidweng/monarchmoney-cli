package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
)

var (
	categoryName    string
	categoryGroupID string
	categoryFile    string
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

		deps, ok := newDeps(renderer, "categories.list", start)
		if !ok {
			return
		}
		svc := deps.Service

		cats, err := svc.ListCategories(cmd.Context())
		if err != nil {
			handleError(renderer, "categories.list", wrapError(err, "failed to list categories"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("categories.list", profile, output.SchemaVersion, "", cats, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-20s %-30s %s\n", "ID", "NAME", "GROUP")
			for _, c := range cats {
				fmt.Printf("%-20s %-30s %s\n", c.ID, c.Name, c.GroupName)
			}
		}
	},
}

var categoriesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a category",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "categories.create", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("categories.create", "", nil, map[string]string{"name": categoryName, "groupId": categoryGroupID})
			env := output.NewEnvelope("categories.create", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		deps, ok := newDeps(renderer, "categories.create", start)
		if !ok {
			return
		}

		result, err := deps.Mutate("categories.create", "", func() (interface{}, error) {
			return deps.Service.CreateCategory(cmd.Context(), categoryName, categoryGroupID)
		}, "failed to create category")
		if err != nil {
			return
		}
		cat := result.(*monarch.Category)

		if jsonMode {
			env := output.NewEnvelope("categories.create", profile, output.SchemaVersion, "", cat, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully created category %s (%s).\n", cat.Name, cat.ID)
		}
	},
}

var categoriesDeleteCmd = &cobra.Command{
	Use:   "delete <category-id>",
	Short: "Delete a category",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		id := args[0]

		if err := safety.Check(safety.TierDestructive, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "categories.delete", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("categories.delete", id, nil, nil)
			env := output.NewEnvelope("categories.delete", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		deps, ok := newDeps(renderer, "categories.delete", start)
		if !ok {
			return
		}

		if _, err := deps.Mutate("categories.delete", id, func() (interface{}, error) {
			return nil, deps.Service.DeleteCategory(cmd.Context(), id)
		}, "failed to delete category"); err != nil {
			return
		}

		if jsonMode {
			env := output.NewEnvelope("categories.delete", profile, output.SchemaVersion, "", map[string]string{"status": "deleted"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully deleted category %s.\n", id)
		}
	},
}

var categoriesDeleteManyCmd = &cobra.Command{
	Use:   "delete-many",
	Short: "Delete multiple categories from a file",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if err := safety.Check(safety.TierDestructive, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "categories.delete-many", err.(*errors.Error), start)
			return
		}

		if categoryFile == "" {
			handleError(renderer, "categories.delete-many", errors.New(errors.InvalidArguments, "--file is required", errors.CatValidation, false, nil), start)
			return
		}

		f, err := os.Open(categoryFile)
		if err != nil {
			handleError(renderer, "categories.delete-many", errors.New(errors.InternalError, "failed to open file", errors.CatInternal, false, err), start)
			return
		}
		defer f.Close()

		var ids []string
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			id := strings.TrimSpace(scanner.Text())
			if id != "" {
				ids = append(ids, id)
			}
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("categories.delete-many", "", nil, map[string]interface{}{"ids": ids})
			env := output.NewEnvelope("categories.delete-many", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		deps, ok := newDeps(renderer, "categories.delete-many", start)
		if !ok {
			return
		}

		if _, err := deps.Mutate("categories.delete-many", "", func() (interface{}, error) {
			return nil, deps.Service.DeleteCategories(cmd.Context(), ids)
		}, "failed to delete categories"); err != nil {
			return
		}

		if jsonMode {
			env := output.NewEnvelope("categories.delete-many", profile, output.SchemaVersion, "", map[string]string{"status": "categories deleted"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully deleted %d categories.\n", len(ids))
		}
	},
}

var categoriesGroupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "List all category groups",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "categories.groups", start)
		if !ok {
			return
		}
		svc := deps.Service

		groups, err := svc.ListCategoryGroups(cmd.Context())
		if err != nil {
			handleError(renderer, "categories.groups", wrapError(err, "failed to list category groups"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("categories.groups", profile, output.SchemaVersion, "", groups, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-20s %-30s %s\n", "ID", "NAME", "TYPE")
			for _, g := range groups {
				fmt.Printf("%-20s %-30s %s\n", g.ID, g.Name, g.Type)
			}
		}
	},
}

func init() {
	categoriesCreateCmd.Flags().StringVar(&categoryName, "name", "", "category name")
	categoriesCreateCmd.Flags().StringVar(&categoryGroupID, "group", "", "category group ID")
	categoriesCreateCmd.MarkFlagRequired("name")
	categoriesCreateCmd.MarkFlagRequired("group")

	categoriesDeleteManyCmd.Flags().StringVar(&categoryFile, "file", "", "file with category IDs (one per line)")
	categoriesDeleteManyCmd.MarkFlagRequired("file")

	categoriesCmd.AddCommand(categoriesListCmd)
	categoriesCmd.AddCommand(categoriesGroupsCmd)
	categoriesCmd.AddCommand(categoriesCreateCmd)
	categoriesCmd.AddCommand(categoriesDeleteCmd)
	categoriesCmd.AddCommand(categoriesDeleteManyCmd)
	RootCmd.AddCommand(categoriesCmd)
}
