package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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

var categoriesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a category",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		logger := audit.NewLogger()

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "categories.create", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("categories.create", "", nil, map[string]string{"name": categoryName, "groupId": categoryGroupID})
			env := output.NewEnvelope("categories.create", profile, "2026-05-08", "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "categories.create", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		cat, err := svc.CreateCategory(cmd.Context(), categoryName, categoryGroupID)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{
			Command:   "categories.create",
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
				cliErr = errors.New(errors.APIError, "failed to create category", errors.CatAPI, false, err)
			}
			handleError(renderer, "categories.create", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("categories.create", profile, "2026-05-08", "", cat, time.Since(start))
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
		logger := audit.NewLogger()
		id := args[0]

		if err := safety.Check(safety.TierDestructive, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "categories.delete", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("categories.delete", id, nil, nil)
			env := output.NewEnvelope("categories.delete", profile, "2026-05-08", "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "categories.delete", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		err = svc.DeleteCategory(cmd.Context(), id)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{
			Command:    "categories.delete",
			ResourceID: id,
			DryRun:     dryRun,
			Confirmed:  confirm,
			Profile:    profile,
			Result:     result,
			ErrorCode:  errCode,
		})

		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to delete category", errors.CatAPI, false, err)
			}
			handleError(renderer, "categories.delete", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("categories.delete", profile, "2026-05-08", "", map[string]string{"status": "deleted"}, time.Since(start))
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
		logger := audit.NewLogger()

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
			env := output.NewEnvelope("categories.delete-many", profile, "2026-05-08", "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "categories.delete-many", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		err = svc.DeleteCategories(cmd.Context(), ids)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{
			Command:   "categories.delete-many",
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
				cliErr = errors.New(errors.APIError, "failed to delete categories", errors.CatAPI, false, err)
			}
			handleError(renderer, "categories.delete-many", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("categories.delete-many", profile, "2026-05-08", "", map[string]string{"status": "categories deleted"}, time.Since(start))
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

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "categories.groups", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		groups, err := svc.ListCategoryGroups(cmd.Context())
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list category groups", errors.CatAPI, false, err)
			}
			handleError(renderer, "categories.groups", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("categories.groups", profile, "2026-05-08", "", groups, time.Since(start))
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
