package cli

import (
	"fmt"
	"strconv"
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
	monthStr     string
	budgetAmount float64
)

var budgetsCmd = &cobra.Command{
	Use:   "budgets",
	Short: "Manage Monarch Money budgets",
}

var budgetsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List budgets",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "budgets.list", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		opts := monarch.ListBudgetsOptions{}
		if monthStr != "" {
			parts := strings.Split(monthStr, "-")
			if len(parts) != 2 {
				handleError(renderer, "budgets.list", errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil), start)
				return
			}
			y, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])
			opts.Year = y
			opts.Month = m
		}

		budgets, err := svc.ListBudgets(cmd.Context(), opts)
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list budgets", errors.CatAPI, false, err)
			}
			handleError(renderer, "budgets.list", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("budgets.list", profile, "2026-05-08", "", budgets, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-30s %10s %10s %10s\n", "CATEGORY", "PLANNED", "ACTUAL", "REMAINING")
			for _, b := range budgets {
				fmt.Printf("%-30s %10.2f %10.2f %10.2f\n", b.CategoryName, b.Planned, b.Actual, b.Planned-b.Actual)
			}
		}
	},
}

var budgetsSetCmd = &cobra.Command{
	Use:   "set <category-id>",
	Short: "Set budget for a category",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		logger := audit.NewLogger()
		categoryID := args[0]

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "budgets.set", err.(*errors.Error), start)
			return
		}

		var y, m int
		if monthStr != "" {
			parts := strings.Split(monthStr, "-")
			if len(parts) != 2 {
				handleError(renderer, "budgets.set", errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil), start)
				return
			}
			y, _ = strconv.Atoi(parts[0])
			m, _ = strconv.Atoi(parts[1])
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("budgets.set", categoryID, nil, map[string]interface{}{"amount": budgetAmount, "month": m, "year": y})
			env := output.NewEnvelope("budgets.set", profile, "2026-05-08", "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "budgets.set", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		budget, err := svc.SetBudget(cmd.Context(), categoryID, budgetAmount, m, y)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{
			Command:    "budgets.set",
			ResourceID: categoryID,
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
				cliErr = errors.New(errors.APIError, "failed to set budget", errors.CatAPI, false, err)
			}
			handleError(renderer, "budgets.set", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("budgets.set", profile, "2026-05-08", "", budget, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully set budget for %s to %.2f.\n", budget.CategoryName, budget.Planned)
		}
	},
}

func init() {
	budgetsListCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	budgetsSetCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	budgetsSetCmd.Flags().Float64Var(&budgetAmount, "amount", 0, "budget amount")
	budgetsSetCmd.MarkFlagRequired("amount")

	budgetsCmd.AddCommand(budgetsListCmd)
	budgetsCmd.AddCommand(budgetsSetCmd)
	RootCmd.AddCommand(budgetsCmd)
}
