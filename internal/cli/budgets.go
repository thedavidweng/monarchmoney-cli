package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
)

var (
	monthStr         string
	budgetAmount     float64
	budgetOverwrite  bool
	budgetCategoryID string
	budgetGroupID    string
	budgetBalance    float64
)

var budgetsCmd = &cobra.Command{
	Use:     "budgets",
	Short:   "Manage Monarch Money budgets",
	GroupID: "core",
	Example: "  monarch budgets list --month 2026-05 --json\n  monarch budgets set --category <id> --amount 500 --confirm",
}

var budgetsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List budgets",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "budgets.list", "failed to list budgets",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.Budget, error) {
				opts := monarch.ListBudgetsOptions{}
				if monthStr != "" {
					parts := strings.Split(monthStr, "-")
					if len(parts) != 2 {
						return nil, errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil)
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
				return svc.ListBudgets(ctx, opts)
			},
			func(budgets []monarch.Budget) {
				fmt.Printf("%-30s %10s %10s %10s\n", "CATEGORY", "PLANNED", "ACTUAL", "REMAINING")
				for _, b := range budgets {
					fmt.Printf("%-30s %10.2f %10.2f %10.2f\n", b.CategoryName, b.Planned, b.Actual, b.Planned-b.Actual)
				}
			})
	},
}

var budgetsSetCmd = &cobra.Command{
	Use:   "set <category-id>",
	Short: "Set budget for a category",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		categoryID := args[0]
		runMutation(cmd, "budgets.set", "failed to set budget", safety.TierMutation, func() (mutation, *errors.Error) {
			var y, m int
			if monthStr != "" {
				parts := strings.Split(monthStr, "-")
				if len(parts) != 2 {
					return mutation{}, errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil)
				}
				y, _ = strconv.Atoi(parts[0])
				m, _ = strconv.Atoi(parts[1])
			} else {
				now := time.Now()
				y = now.Year()
				m = int(now.Month())
			}
			var budget *monarch.Budget
			return mutation{
				resourceID: categoryID,
				planAfter:  map[string]any{"amount": budgetAmount, "month": m, "year": y},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					b, err := svc.SetBudget(ctx, categoryID, budgetAmount, fmt.Sprintf("%04d-%02d-01", y, m))
					if err != nil {
						return nil, err
					}
					budget = b
					return b, nil
				},
				human: func() {
					fmt.Printf("Successfully set budget for %s to %.2f.\n", categoryID, budget.Planned)
				},
			}, nil
		})
	},
}

var budgetsResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset budget for a month",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "budgets.reset", "failed to reset budget", safety.TierDestructive, func() (mutation, *errors.Error) {
			startDate, verr := budgetMonthStart()
			if verr != nil {
				return mutation{}, verr
			}
			var categoryIDs []string
			if budgetCategoryID != "" {
				categoryIDs = []string{budgetCategoryID}
			}
			return mutation{
				planAfter: map[string]any{"month": startDate, "overwrite": budgetOverwrite, "categories": categoryIDs},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.ResetBudget(ctx, startDate, budgetOverwrite, categoryIDs); err != nil {
						return nil, err
					}
					return map[string]string{"status": "budget reset"}, nil
				},
				human: func() { fmt.Printf("Successfully reset budget for %s.\n", startDate) },
			}, nil
		})
	},
}

func budgetMonthStart() (string, *errors.Error) {
	if monthStr == "" {
		return "", errors.New(errors.InvalidArguments, "--month is required (YYYY-MM)", errors.CatValidation, false, nil)
	}
	parts := strings.Split(monthStr, "-")
	if len(parts) != 2 {
		return "", errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil)
	}
	y, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return fmt.Sprintf("%04d-%02d-01", y, m), nil
}

var budgetsSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Show budget system settings",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "budgets.settings", "failed to get budget settings",
			func(ctx context.Context, svc *monarch.Service) (*monarch.BudgetSettings, error) {
				return svc.GetBudgetSettings(ctx)
			},
			func(settings *monarch.BudgetSettings) {
				fmt.Printf("System:                    %s\n", settings.System)
				fmt.Printf("Has budget:                %t\n", settings.HasBudget)
				fmt.Printf("Has transactions:          %t\n", settings.HasTransactions)
			})
	},
}

var budgetsSetGroupCmd = &cobra.Command{
	Use:   "set-group <group-id>",
	Short: "Set budget amount for a category group",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		groupID := args[0]
		runMutation(cmd, "budgets.set-group", "failed to set group budget", safety.TierMutation, func() (mutation, *errors.Error) {
			startDate, verr := budgetMonthStartOrCurrent()
			if verr != nil {
				return mutation{}, verr
			}
			return mutation{
				resourceID: groupID,
				planAfter:  map[string]any{"amount": budgetAmount, "month": startDate},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.SetBudgetGroup(ctx, groupID, budgetAmount, startDate); err != nil {
						return nil, err
					}
					return map[string]string{"status": "group budget set"}, nil
				},
				human: func() {
					fmt.Printf("Successfully set budget for group %s to %.2f.\n", groupID, budgetAmount)
				},
			}, nil
		})
	},
}

func budgetMonthStartOrCurrent() (string, *errors.Error) {
	if monthStr == "" {
		now := time.Now()
		return fmt.Sprintf("%04d-%02d-01", now.Year(), now.Month()), nil
	}
	return budgetMonthStart()
}

var budgetsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a budget for a month",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "budgets.create", "failed to create budget", safety.TierMutation, func() (mutation, *errors.Error) {
			startDate, verr := budgetMonthStart()
			if verr != nil {
				return mutation{}, verr
			}
			return mutation{
				planAfter: map[string]string{"month": startDate},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.CreateBudget(ctx, startDate); err != nil {
						return nil, err
					}
					return map[string]string{"status": "budget created"}, nil
				},
				human: func() { fmt.Printf("Successfully created budget for %s.\n", startDate) },
			}, nil
		})
	},
}

var budgetsClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all budget amounts for a month",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "budgets.clear", "failed to clear budget", safety.TierDestructive, func() (mutation, *errors.Error) {
			startDate, verr := budgetMonthStart()
			if verr != nil {
				return mutation{}, verr
			}
			return mutation{
				planAfter: map[string]string{"month": startDate},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.ClearBudget(ctx, startDate); err != nil {
						return nil, err
					}
					return map[string]string{"status": "budget cleared"}, nil
				},
				human: func() { fmt.Printf("Successfully cleared budget for %s.\n", startDate) },
			}, nil
		})
	},
}

var budgetsResetRolloverCmd = &cobra.Command{
	Use:   "reset-rollover",
	Short: "Reset rollover for a category or group",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "budgets.reset-rollover", "failed to reset budget rollover", safety.TierMutation, func() (mutation, *errors.Error) {
			startDate, verr := budgetMonthStart()
			if verr != nil {
				return mutation{}, verr
			}
			if (budgetCategoryID == "") == (budgetGroupID == "") {
				return mutation{}, errors.New(errors.InvalidArguments, "pass exactly one of --category-id, --group-id", errors.CatValidation, false, nil)
			}
			opts := &monarch.ResetRolloverOptions{
				StartMonth:      startDate,
				CategoryID:      budgetCategoryID,
				CategoryGroupID: budgetGroupID,
			}
			after := map[string]any{"month": startDate, "category": budgetCategoryID, "group": budgetGroupID}
			if cmd.Flags().Changed("balance") {
				opts.StartingBalance = &budgetBalance
				after["balance"] = budgetBalance
			}
			return mutation{
				planAfter: after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.ResetBudgetRollover(ctx, opts); err != nil {
						return nil, err
					}
					return map[string]string{"status": "rollover reset"}, nil
				},
				human: func() { fmt.Printf("Successfully reset rollover for %s.\n", startDate) },
			}, nil
		})
	},
}

var budgetsFlexRolloverShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show flexible budget rollover settings",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "budgets.flex-rollover.show", "failed to get flex rollover settings",
			func(ctx context.Context, svc *monarch.Service) (*monarch.FlexRolloverPeriod, error) {
				return svc.GetFlexRolloverSettings(ctx)
			},
			func(period *monarch.FlexRolloverPeriod) {
				fmt.Printf("Start month:      %s\n", period.StartMonth)
				fmt.Printf("Starting balance: %.2f\n", period.StartingBalance)
				fmt.Printf("Frequency:        %s\n", period.Frequency)
			})
	},
}

var budgetsShowCmd = &cobra.Command{
	Use:   "show <category-id>",
	Short: "Show budget for a specific category",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		categoryID := args[0]
		run(cmd.Context(), "budgets.show", "failed to get budget",
			func(ctx context.Context, svc *monarch.Service) (*monarch.Budget, error) {
				var y, m int
				if monthStr != "" {
					parts := strings.Split(monthStr, "-")
					if len(parts) != 2 {
						return nil, errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil)
					}
					y, _ = strconv.Atoi(parts[0])
					m, _ = strconv.Atoi(parts[1])
				} else {
					now := time.Now()
					y = now.Year()
					m = int(now.Month())
				}
				startDate := fmt.Sprintf("%04d-%02d-01", y, m)
				endDate := time.Date(y, time.Month(m+1), 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
				return svc.GetBudget(ctx, categoryID, startDate, endDate)
			},
			func(budget *monarch.Budget) {
				fmt.Printf("Category:  %s\n", budget.CategoryName)
				fmt.Printf("Planned:   %.2f\n", budget.Planned)
				fmt.Printf("Actual:    %.2f\n", budget.Actual)
				fmt.Printf("Remaining: %.2f\n", budget.Planned-budget.Actual)
			})
	},
}

var budgetsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export budgets",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "budgets.export", "failed to list budgets",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.Budget, error) {
				opts := monarch.ListBudgetsOptions{}
				if monthStr != "" {
					parts := strings.Split(monthStr, "-")
					if len(parts) != 2 {
						return nil, errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil)
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
				return svc.ListBudgets(ctx, opts)
			},
			func(_ []monarch.Budget) {})
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
		runMutation(cmd, "budgets.flexible.set", "failed to set flexible budget", safety.TierMutation, func() (mutation, *errors.Error) {
			var y, m int
			if monthStr != "" {
				parts := strings.Split(monthStr, "-")
				if len(parts) != 2 {
					return mutation{}, errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil)
				}
				y, _ = strconv.Atoi(parts[0])
				m, _ = strconv.Atoi(parts[1])
			} else {
				now := time.Now()
				y = now.Year()
				m = int(now.Month())
			}
			return mutation{
				resourceID: fmt.Sprintf("%d-%02d", y, m),
				planAfter:  map[string]any{"amount": budgetAmount},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.UpdateFlexibleBudget(ctx, m, y, budgetAmount); err != nil {
						return nil, err
					}
					return map[string]string{"status": "budget set"}, nil
				},
				human: func() {
					fmt.Printf("Successfully set flexible budget for %d-%02d to %.2f.\n", y, m, budgetAmount)
				},
			}, nil
		})
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
		runMutation(cmd, "budgets.flex-rollover.set", "failed to set flex rollover", safety.TierMutation, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: monthStr,
				planAfter:  map[string]any{"balance": budgetAmount},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.UpdateFlexRolloverSettings(ctx, monthStr, budgetAmount, true); err != nil {
						return nil, err
					}
					return map[string]string{"status": "rollover set"}, nil
				},
				human: func() {
					fmt.Printf("Successfully set flex rollover starting %s with balance %.2f.\n", monthStr, budgetAmount)
				},
			}, nil
		})
	},
}

func init() {
	budgetsListCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")

	budgetsShowCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")

	budgetsSetCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	budgetsSetCmd.Flags().Float64Var(&budgetAmount, "amount", 0, "budget amount")
	budgetsSetCmd.MarkFlagRequired("amount") //nolint:errcheck // flag registered above

	budgetsResetCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	budgetsResetCmd.Flags().BoolVar(&budgetOverwrite, "overwrite", false, "overwrite existing budget amounts")
	budgetsResetCmd.Flags().StringVar(&budgetCategoryID, "category-id", "", "only reset these categories (repeatable via comma)")
	budgetsResetCmd.MarkFlagRequired("month") //nolint:errcheck // flag registered above

	budgetsSetGroupCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	budgetsSetGroupCmd.Flags().Float64Var(&budgetAmount, "amount", 0, "budget amount")
	budgetsSetGroupCmd.MarkFlagRequired("amount") //nolint:errcheck // flag registered above

	budgetsCreateCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	budgetsCreateCmd.MarkFlagRequired("month") //nolint:errcheck // flag registered above

	budgetsClearCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	budgetsClearCmd.MarkFlagRequired("month") //nolint:errcheck // flag registered above

	budgetsResetRolloverCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	budgetsResetRolloverCmd.Flags().StringVar(&budgetCategoryID, "category-id", "", "category ID")
	budgetsResetRolloverCmd.Flags().StringVar(&budgetGroupID, "group-id", "", "category group ID")
	budgetsResetRolloverCmd.Flags().Float64Var(&budgetBalance, "balance", 0, "starting balance")
	budgetsResetRolloverCmd.MarkFlagRequired("month") //nolint:errcheck // flag registered above

	budgetsExportCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")

	budgetsFlexibleSetCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	budgetsFlexibleSetCmd.Flags().Float64Var(&budgetAmount, "amount", 0, "budget amount")
	budgetsFlexibleSetCmd.MarkFlagRequired("amount") //nolint:errcheck // flag registered above
	budgetsFlexibleCmd.AddCommand(budgetsFlexibleSetCmd)

	budgetsFlexRolloverSetCmd.Flags().StringVar(&monthStr, "month", "", "start month in YYYY-MM-DD format")
	budgetsFlexRolloverSetCmd.Flags().Float64Var(&budgetAmount, "amount", 0, "starting balance")
	budgetsFlexRolloverSetCmd.MarkFlagRequired("month") //nolint:errcheck // flag registered above
	budgetsFlexRolloverCmd.AddCommand(budgetsFlexRolloverSetCmd)
	budgetsFlexRolloverCmd.AddCommand(budgetsFlexRolloverShowCmd)

	budgetsCmd.AddCommand(budgetsListCmd)
	budgetsCmd.AddCommand(budgetsShowCmd)
	budgetsCmd.AddCommand(budgetsSettingsCmd)
	budgetsCmd.AddCommand(budgetsSetCmd)
	budgetsCmd.AddCommand(budgetsSetGroupCmd)
	budgetsCmd.AddCommand(budgetsCreateCmd)
	budgetsCmd.AddCommand(budgetsClearCmd)
	budgetsCmd.AddCommand(budgetsResetCmd)
	budgetsCmd.AddCommand(budgetsResetRolloverCmd)
	budgetsCmd.AddCommand(budgetsExportCmd)
	budgetsCmd.AddCommand(budgetsFlexibleCmd)
	budgetsCmd.AddCommand(budgetsFlexRolloverCmd)
	RootCmd.AddCommand(budgetsCmd)
}
