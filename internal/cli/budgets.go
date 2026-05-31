package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
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

		deps, ok := newDeps(renderer, "budgets.list", start)
		if !ok {
			return
		}
		svc := deps.Service

		opts := monarch.ListBudgetsOptions{}
		if monthStr != "" {
			parts := strings.Split(monthStr, "-")
			if len(parts) != 2 {
				handleError(renderer, "budgets.list", errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil), start)
				return
			}
			y, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])
			// Convert YYYY-MM to startDate/endDate format
			opts.StartDate = fmt.Sprintf("%04d-%02d-01", y, m)
			// Last day of month
			lastDay := time.Date(y, time.Month(m+1), 0, 0, 0, 0, 0, time.UTC).Day()
			opts.EndDate = fmt.Sprintf("%04d-%02d-%02d", y, m, lastDay)
		} else {
			// Default to current month
			now := time.Now()
			opts.StartDate = fmt.Sprintf("%04d-%02d-01", now.Year(), now.Month())
			lastDay := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
			opts.EndDate = fmt.Sprintf("%04d-%02d-%02d", now.Year(), now.Month(), lastDay)
		}

		budgets, err := svc.ListBudgets(cmd.Context(), opts)
		if err != nil {
			handleError(renderer, "budgets.list", wrapError(err, "failed to list budgets"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("budgets.list", profile, output.SchemaVersion, "", budgets, time.Since(start))
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
		} else {
			now := time.Now()
			y = now.Year()
			m = int(now.Month())
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("budgets.set", categoryID, nil, map[string]interface{}{"amount": budgetAmount, "month": m, "year": y})
			env := output.NewEnvelope("budgets.set", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		deps, ok := newDeps(renderer, "budgets.set", start)
		if !ok {
			return
		}

		result, err := deps.Mutate("budgets.set", categoryID, func() (interface{}, error) {
			return deps.Service.SetBudget(cmd.Context(), categoryID, budgetAmount, fmt.Sprintf("%04d-%02d-01", y, m))
		}, "failed to set budget")
		if err != nil {
			return
		}
		budget := result.(*monarch.Budget)

		if jsonMode {
			env := output.NewEnvelope("budgets.set", profile, output.SchemaVersion, "", budget, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully set budget for %s to %.2f.\n", budget.CategoryName, budget.Planned)
		}
	},
}

var budgetsResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset budget for a month",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if err := safety.Check(safety.TierDestructive, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "budgets.reset", err.(*errors.Error), start)
			return
		}

		var y, m int
		if monthStr == "" {
			handleError(renderer, "budgets.reset", errors.New(errors.InvalidArguments, "--month is required", errors.CatValidation, false, nil), start)
			return
		}
		parts := strings.Split(monthStr, "-")
		if len(parts) != 2 {
			handleError(renderer, "budgets.reset", errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil), start)
			return
		}
		y, _ = strconv.Atoi(parts[0])
		m, _ = strconv.Atoi(parts[1])

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("budgets.reset", "", nil, map[string]int{"month": m, "year": y})
			env := output.NewEnvelope("budgets.reset", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		deps, ok := newDeps(renderer, "budgets.reset", start)
		if !ok {
			return
		}

		if _, err := deps.Mutate("budgets.reset", "", func() (interface{}, error) {
			return nil, deps.Service.ResetBudget(cmd.Context(), m, y)
		}, "failed to reset budget"); err != nil {
			return
		}

		if jsonMode {
			env := output.NewEnvelope("budgets.reset", profile, output.SchemaVersion, "", map[string]string{"status": "budget reset"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully reset budget for %d-%02d.\n", y, m)
		}
	},
}

var budgetsShowCmd = &cobra.Command{
	Use:   "show <category-id>",
	Short: "Show budget for a specific category",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		categoryID := args[0]

		var y, m int
		if monthStr != "" {
			parts := strings.Split(monthStr, "-")
			if len(parts) != 2 {
				handleError(renderer, "budgets.show", errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil), start)
				return
			}
			y, _ = strconv.Atoi(parts[0])
			m, _ = strconv.Atoi(parts[1])
		} else {
			now := time.Now()
			y = now.Year()
			m = int(now.Month())
		}

		deps, ok := newDeps(renderer, "budgets.show", start)
		if !ok {
			return
		}
		svc := deps.Service

		startDate := fmt.Sprintf("%04d-%02d-01", y, m)
		endDate := time.Date(y, time.Month(m+1), 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		budget, err := svc.GetBudget(cmd.Context(), categoryID, startDate, endDate)
		if err != nil {
			handleError(renderer, "budgets.show", wrapError(err, "failed to get budget"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("budgets.show", profile, output.SchemaVersion, "", budget, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Category:  %s\n", budget.CategoryName)
			fmt.Printf("Planned:   %.2f\n", budget.Planned)
			fmt.Printf("Actual:    %.2f\n", budget.Actual)
			fmt.Printf("Remaining: %.2f\n", budget.Planned-budget.Actual)
		}
	},
}

var budgetsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export budgets",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "budgets.export", start)
		if !ok {
			return
		}
		svc := deps.Service

		opts := monarch.ListBudgetsOptions{}
		if monthStr != "" {
			parts := strings.Split(monthStr, "-")
			if len(parts) != 2 {
				handleError(renderer, "budgets.export", errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil), start)
				return
			}
			y, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])
			opts.StartDate = fmt.Sprintf("%04d-%02d-01", y, m)
			lastDay := time.Date(y, time.Month(m+1), 0, 0, 0, 0, 0, time.UTC).Day()
			opts.EndDate = fmt.Sprintf("%04d-%02d-%02d", y, m, lastDay)
		} else {
			now := time.Now()
			opts.StartDate = fmt.Sprintf("%04d-%02d-01", now.Year(), now.Month())
			lastDay := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
			opts.EndDate = fmt.Sprintf("%04d-%02d-%02d", now.Year(), now.Month(), lastDay)
		}

		budgets, err := svc.ListBudgets(cmd.Context(), opts)
		if err != nil {
			handleError(renderer, "budgets.export", wrapError(err, "failed to list budgets"), start)
			return
		}

		env := output.NewEnvelope("budgets.export", profile, output.SchemaVersion, "", budgets, time.Since(start))
		renderer.RenderSuccess(env)
	},
}

var budgetsFlexibleCmd = &cobra.Command{
	Use:   "flexible",
	Short: "Manage flexible budget settings",
}

var budgetsFlexibleSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set flexible budget amount for a month",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "budgets.flexible.set", err.(*errors.Error), start)
			return
		}

		var y, m int
		if monthStr != "" {
			parts := strings.Split(monthStr, "-")
			if len(parts) != 2 {
				handleError(renderer, "budgets.flexible.set", errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil), start)
				return
			}
			y, _ = strconv.Atoi(parts[0])
			m, _ = strconv.Atoi(parts[1])
		} else {
			now := time.Now()
			y = now.Year()
			m = int(now.Month())
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("budgets.flexible.set", fmt.Sprintf("%d-%02d", y, m), nil, map[string]interface{}{"amount": budgetAmount})
			env := output.NewEnvelope("budgets.flexible.set", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		deps, ok := newDeps(renderer, "budgets.flexible.set", start)
		if !ok {
			return
		}

		if _, err := deps.Mutate("budgets.flexible.set", fmt.Sprintf("%d-%02d", y, m), func() (interface{}, error) {
			return nil, deps.Service.UpdateFlexibleBudget(cmd.Context(), m, y, budgetAmount)
		}, "failed to set flexible budget"); err != nil {
			return
		}

		if jsonMode {
			env := output.NewEnvelope("budgets.flexible.set", profile, output.SchemaVersion, "", map[string]string{"status": "budget set"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully set flexible budget for %d-%02d to %.2f.\n", y, m, budgetAmount)
		}
	},
}

var budgetsFlexRolloverCmd = &cobra.Command{
	Use:   "flex-rollover",
	Short: "Manage flexible budget rollover settings",
}

var budgetsFlexRolloverSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set flexible budget rollover settings",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "budgets.flex-rollover.set", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("budgets.flex-rollover.set", monthStr, nil, map[string]interface{}{"balance": budgetAmount})
			env := output.NewEnvelope("budgets.flex-rollover.set", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		deps, ok := newDeps(renderer, "budgets.flex-rollover.set", start)
		if !ok {
			return
		}

		if _, err := deps.Mutate("budgets.flex-rollover.set", monthStr, func() (interface{}, error) {
			return nil, deps.Service.UpdateFlexRolloverSettings(cmd.Context(), monthStr, budgetAmount, true)
		}, "failed to set flex rollover"); err != nil {
			return
		}

		if jsonMode {
			env := output.NewEnvelope("budgets.flex-rollover.set", profile, output.SchemaVersion, "", map[string]string{"status": "rollover set"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully set flex rollover starting %s with balance %.2f.\n", monthStr, budgetAmount)
		}
	},
}

func init() {
	budgetsListCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")

	budgetsShowCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")

	budgetsSetCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	budgetsSetCmd.Flags().Float64Var(&budgetAmount, "amount", 0, "budget amount")
	budgetsSetCmd.MarkFlagRequired("amount")

	budgetsResetCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	budgetsResetCmd.MarkFlagRequired("month")

	budgetsExportCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")

	budgetsFlexibleSetCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	budgetsFlexibleSetCmd.Flags().Float64Var(&budgetAmount, "amount", 0, "budget amount")
	budgetsFlexibleSetCmd.MarkFlagRequired("amount")
	budgetsFlexibleCmd.AddCommand(budgetsFlexibleSetCmd)

	budgetsFlexRolloverSetCmd.Flags().StringVar(&monthStr, "month", "", "start month in YYYY-MM-DD format")
	budgetsFlexRolloverSetCmd.Flags().Float64Var(&budgetAmount, "amount", 0, "starting balance")
	budgetsFlexRolloverSetCmd.MarkFlagRequired("month")
	budgetsFlexRolloverCmd.AddCommand(budgetsFlexRolloverSetCmd)

	budgetsCmd.AddCommand(budgetsListCmd)
	budgetsCmd.AddCommand(budgetsShowCmd)
	budgetsCmd.AddCommand(budgetsSetCmd)
	budgetsCmd.AddCommand(budgetsResetCmd)
	budgetsCmd.AddCommand(budgetsExportCmd)
	budgetsCmd.AddCommand(budgetsFlexibleCmd)
	budgetsCmd.AddCommand(budgetsFlexRolloverCmd)
	RootCmd.AddCommand(budgetsCmd)
}
