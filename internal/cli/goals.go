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
	goalName          string
	goalType          string
	goalTargetAmount  float64
	goalTargetDate    string
	goalMonthly       float64
	goalSinking       bool
	goalPriority      int
	goalPriorityIDs   []string
	goalAccountID     string
	goalAmount        float64
	goalDate          string
	goalNotes         string
	goalEntireBalance bool
	goalApplyFuture   bool
)

var goalsCmd = &cobra.Command{
	Use:     "goals",
	Short:   "Manage Monarch Money goals",
	GroupID: "core",
	Example: "  monarch goals list --json\n  monarch goals budgets --month 2026-05",
}

var goalsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List goals",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "goals.list", "failed to list goals",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.Goal, error) {
				return svc.ListGoals(ctx)
			},
			func(goals []monarch.Goal) {
				fmt.Printf("%-20s %-20s %-10s %12s %12s %12s\n", "ID", "NAME", "STATUS", "BALANCE", "TARGET", "PROGRESS")
				for _, g := range goals {
					fmt.Printf("%-20s %-20s %-10s %12.2f %12.2f %11.1f%%\n", g.ID, g.Name, g.Status, g.CurrentBalance, g.TargetAmount, g.Progress*100)
				}
			})
	},
}

var goalsBudgetsCmd = &cobra.Command{
	Use:   "budgets",
	Short: "List savings goal monthly budget amounts",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "goals.budgets", "failed to list goal budgets",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.SavingsGoalBudget, error) {
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
				return svc.ListSavingsGoalBudgets(ctx, startDate, endDate)
			},
			func(budgets []monarch.SavingsGoalBudget) {
				fmt.Printf("%-20s %-20s %-10s %10s %10s %10s\n", "GOAL", "MONTH", "STATUS", "PLANNED", "ACTUAL", "REMAINING")
				for _, b := range budgets {
					fmt.Printf("%-20s %-20s %-10s %10.2f %10.2f %10.2f\n", b.GoalName, b.Month, b.GoalStatus, b.Planned, b.Actual, b.Remaining)
				}
			})
	},
}

var goalsShowCmd = &cobra.Command{
	Use:   "show <goal-id>",
	Short: "Show a goal",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "goals.show", "failed to get goal",
			func(ctx context.Context, svc *monarch.Service) (*monarch.Goal, error) {
				return svc.GetGoal(ctx, args[0])
			},
			func(g *monarch.Goal) {
				fmt.Printf("ID:       %s\n", g.ID)
				fmt.Printf("Name:     %s\n", g.Name)
				fmt.Printf("Status:   %s\n", g.Status)
				fmt.Printf("Balance:  %.2f / %.2f (%.1f%%)\n", g.CurrentBalance, g.TargetAmount, g.Progress*100)
				fmt.Printf("Target:   %s\n", g.TargetDate)
			})
	},
}

var goalsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a savings goal",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "goals.create", "failed to create goal", safety.TierMutation, func() (mutation, *errors.Error) {
			input := &monarch.GoalInput{Name: goalName}
			after := map[string]any{"name": goalName}
			if cmd.Flags().Changed("type") {
				input.Type = goalType
				after["type"] = goalType
			}
			if cmd.Flags().Changed("target-amount") {
				input.TargetAmount = &goalTargetAmount
				after["target_amount"] = goalTargetAmount
			}
			if cmd.Flags().Changed("target-date") {
				input.TargetDate = &goalTargetDate
				after["target_date"] = goalTargetDate
			}
			if cmd.Flags().Changed("monthly") {
				input.MonthlyContribution = &goalMonthly
				after["monthly"] = goalMonthly
			}
			if cmd.Flags().Changed("sinking-fund") {
				input.SinkingFund = &goalSinking
				after["sinking_fund"] = goalSinking
			}
			if cmd.Flags().Changed("priority") {
				input.Priority = &goalPriority
				after["priority"] = goalPriority
			}
			var goal *monarch.Goal
			return mutation{
				planAfter: after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					created, err := svc.CreateGoal(ctx, input)
					if err != nil {
						return nil, err
					}
					goal = created
					return created, nil
				},
				human: func() { fmt.Printf("Successfully created goal %s (%s).\n", goal.Name, goal.ID) },
			}, nil
		})
	},
}

var goalsUpdateCmd = &cobra.Command{
	Use:   "update <goal-id>",
	Short: "Update a savings goal",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "goals.update", "failed to update goal", safety.TierMutation, func() (mutation, *errors.Error) {
			update := &monarch.GoalUpdate{}
			after := map[string]any{}
			if cmd.Flags().Changed("name") {
				update.Name = &goalName
				after["name"] = goalName
			}
			if cmd.Flags().Changed("type") {
				update.Type = &goalType
				after["type"] = goalType
			}
			if cmd.Flags().Changed("target-amount") {
				update.TargetAmount = &goalTargetAmount
				after["target_amount"] = goalTargetAmount
			}
			if cmd.Flags().Changed("target-date") {
				update.TargetDate = &goalTargetDate
				after["target_date"] = goalTargetDate
			}
			if cmd.Flags().Changed("monthly") {
				update.MonthlyContribution = &goalMonthly
				after["monthly"] = goalMonthly
			}
			if cmd.Flags().Changed("sinking-fund") {
				update.SinkingFund = &goalSinking
				after["sinking_fund"] = goalSinking
			}
			if cmd.Flags().Changed("priority") {
				update.Priority = &goalPriority
				after["priority"] = goalPriority
			}
			if len(after) == 0 {
				return mutation{}, errors.New(errors.InvalidArguments, "at least one update flag is required", errors.CatValidation, false, nil)
			}
			var goal *monarch.Goal
			return mutation{
				resourceID: id,
				planAfter:  after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateGoal(ctx, id, update)
					if err != nil {
						return nil, err
					}
					goal = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Successfully updated goal %s (%s).\n", goal.Name, goal.ID) },
			}, nil
		})
	},
}

var goalsDeleteCmd = &cobra.Command{
	Use:   "delete <goal-id>",
	Short: "Delete a savings goal",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "goals.delete", "failed to delete goal", safety.TierDestructive, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.DeleteGoal(ctx, id); err != nil {
						return nil, err
					}
					return map[string]string{"status": "deleted"}, nil
				},
				human: func() { fmt.Printf("Successfully deleted goal %s.\n", id) },
			}, nil
		})
	},
}

var goalsArchiveCmd = &cobra.Command{
	Use:   "archive <goal-id>",
	Short: "Archive a savings goal",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "goals.archive", "failed to archive goal", safety.TierMutation, func() (mutation, *errors.Error) {
			var goal *monarch.Goal
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					archived, err := svc.ArchiveGoal(ctx, id)
					if err != nil {
						return nil, err
					}
					goal = archived
					return archived, nil
				},
				human: func() { fmt.Printf("Successfully archived goal %s (%s).\n", goal.Name, goal.ID) },
			}, nil
		})
	},
}

var goalsRestoreCmd = &cobra.Command{
	Use:   "restore <goal-id>",
	Short: "Restore an archived savings goal",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "goals.restore", "failed to restore goal", safety.TierMutation, func() (mutation, *errors.Error) {
			var goal *monarch.Goal
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					restored, err := svc.RestoreGoal(ctx, id)
					if err != nil {
						return nil, err
					}
					goal = restored
					return restored, nil
				},
				human: func() { fmt.Printf("Successfully restored goal %s (%s).\n", goal.Name, goal.ID) },
			}, nil
		})
	},
}

var goalsPrioritiesCmd = &cobra.Command{
	Use:   "priorities",
	Short: "Set goal priority order (first --id is highest)",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "goals.priorities", "failed to update goal priorities", safety.TierMutation, func() (mutation, *errors.Error) {
			return mutation{
				planAfter: map[string]any{"ids": goalPriorityIDs},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.UpdateGoalPriorities(ctx, goalPriorityIDs); err != nil {
						return nil, err
					}
					return map[string]string{"status": "priorities updated"}, nil
				},
				human: func() { fmt.Println("Successfully updated goal priorities.") },
			}, nil
		})
	},
}

var goalsLinkAccountCmd = &cobra.Command{
	Use:   "link-account <goal-id>",
	Short: "Link an account balance to a goal",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "goals.link-account", "failed to link goal account", safety.TierMutation, func() (mutation, *errors.Error) {
			var amount *float64
			after := map[string]any{"account": goalAccountID, "entire_balance": goalEntireBalance}
			if cmd.Flags().Changed("amount") {
				amount = &goalAmount
				after["amount"] = goalAmount
			}
			var goal *monarch.Goal
			return mutation{
				resourceID: id,
				planAfter:  after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					linked, err := svc.LinkGoalAccount(ctx, id, goalAccountID, amount, goalEntireBalance)
					if err != nil {
						return nil, err
					}
					goal = linked
					return linked, nil
				},
				human: func() { fmt.Printf("Successfully linked account to goal %s.\n", goal.ID) },
			}, nil
		})
	},
}

var goalsUnlinkAccountCmd = &cobra.Command{
	Use:   "unlink-account <goal-id>",
	Short: "Unlink an account balance from a goal",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "goals.unlink-account", "failed to unlink goal account", safety.TierMutation, func() (mutation, *errors.Error) {
			var goal *monarch.Goal
			return mutation{
				resourceID: id,
				planAfter:  map[string]string{"account": goalAccountID},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					unlinked, err := svc.UnlinkGoalAccount(ctx, id, goalAccountID)
					if err != nil {
						return nil, err
					}
					goal = unlinked
					return unlinked, nil
				},
				human: func() { fmt.Printf("Successfully unlinked account from goal %s.\n", goal.ID) },
			}, nil
		})
	},
}

var goalsEventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Manage goal events (contributions and withdrawals)",
}

var goalsEventsListCmd = &cobra.Command{
	Use:   "list <goal-id>",
	Short: "List events for a goal",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "goals.events.list", "failed to list goal events",
			func(ctx context.Context, svc *monarch.Service) ([]*monarch.GoalEvent, error) {
				return svc.ListGoalEvents(ctx, args[0])
			},
			func(events []*monarch.GoalEvent) {
				fmt.Printf("%-20s %-12s %10s %-10s %s\n", "ID", "DATE", "AMOUNT", "TYPE", "NOTES")
				for _, e := range events {
					fmt.Printf("%-20s %-12s %10.2f %-10s %s\n", e.ID, e.Date, e.Amount, e.Type, e.Notes)
				}
			})
	},
}

var goalsEventsContributeCmd = &cobra.Command{
	Use:   "contribute <goal-id>",
	Short: "Contribute to a goal from an account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "goals.events.contribute", "failed to contribute to goal", safety.TierMutation, func() (mutation, *errors.Error) {
			var date, notes *string
			after := map[string]any{"account": goalAccountID, "amount": goalAmount}
			if cmd.Flags().Changed("date") {
				date = &goalDate
				after["date"] = goalDate
			}
			if cmd.Flags().Changed("notes") {
				notes = &goalNotes
				after["notes"] = goalNotes
			}
			var event *monarch.GoalEvent
			return mutation{
				resourceID: id,
				planAfter:  after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					created, err := svc.ContributeToGoal(ctx, id, goalAccountID, goalAmount, date, notes)
					if err != nil {
						return nil, err
					}
					event = created
					return created, nil
				},
				human: func() { fmt.Printf("Contributed %.2f to goal %s (event %s).\n", event.Amount, id, event.ID) },
			}, nil
		})
	},
}

var goalsEventsWithdrawCmd = &cobra.Command{
	Use:   "withdraw <goal-id>",
	Short: "Withdraw from a goal to an account",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "goals.events.withdraw", "failed to withdraw from goal", safety.TierMutation, func() (mutation, *errors.Error) {
			var date, notes *string
			after := map[string]any{"account": goalAccountID, "amount": goalAmount}
			if cmd.Flags().Changed("date") {
				date = &goalDate
				after["date"] = goalDate
			}
			if cmd.Flags().Changed("notes") {
				notes = &goalNotes
				after["notes"] = goalNotes
			}
			var event *monarch.GoalEvent
			return mutation{
				resourceID: id,
				planAfter:  after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					created, err := svc.WithdrawFromGoal(ctx, id, goalAccountID, goalAmount, date, notes)
					if err != nil {
						return nil, err
					}
					event = created
					return created, nil
				},
				human: func() { fmt.Printf("Withdrew %.2f from goal %s (event %s).\n", event.Amount, id, event.ID) },
			}, nil
		})
	},
}

var goalsEventsUpdateCmd = &cobra.Command{
	Use:   "update <event-id>",
	Short: "Update a goal event",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "goals.events.update", "failed to update goal event", safety.TierMutation, func() (mutation, *errors.Error) {
			var date, notes *string
			after := map[string]any{}
			if cmd.Flags().Changed("date") {
				date = &goalDate
				after["date"] = goalDate
			}
			if cmd.Flags().Changed("notes") {
				notes = &goalNotes
				after["notes"] = goalNotes
			}
			if len(after) == 0 {
				return mutation{}, errors.New(errors.InvalidArguments, "at least one of --date, --notes is required", errors.CatValidation, false, nil)
			}
			var event *monarch.GoalEvent
			return mutation{
				resourceID: id,
				planAfter:  after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateGoalEvent(ctx, id, date, notes)
					if err != nil {
						return nil, err
					}
					event = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Successfully updated goal event %s.\n", event.ID) },
			}, nil
		})
	},
}

var goalsEventsDeleteCmd = &cobra.Command{
	Use:   "delete <event-id>",
	Short: "Delete a goal event",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "goals.events.delete", "failed to delete goal event", safety.TierDestructive, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.DeleteGoalEvent(ctx, id); err != nil {
						return nil, err
					}
					return map[string]string{"status": "deleted"}, nil
				},
				human: func() { fmt.Printf("Successfully deleted goal event %s.\n", id) },
			}, nil
		})
	},
}

var goalsBudgetCmd = &cobra.Command{
	Use:   "budget <goal-id>",
	Short: "Show monthly budget amounts for a goal",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "goals.budget", "failed to get goal budget amounts",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.GoalBudgetAmount, error) {
				start, end, verr := goalsBudgetRange()
				if verr != nil {
					return nil, verr
				}
				return svc.GetGoalBudgetAmounts(ctx, args[0], start, end)
			},
			func(amounts []monarch.GoalBudgetAmount) {
				fmt.Printf("%-12s %10s %10s %10s\n", "MONTH", "PLANNED", "ACTUAL", "REMAINING")
				for _, a := range amounts {
					fmt.Printf("%-12s %10.2f %10.2f %10.2f\n", a.Month, a.Planned, a.Actual, a.Remaining)
				}
			})
	},
}

func goalsBudgetRange() (start, end string, verr *errors.Error) {
	var y, m int
	if monthStr != "" {
		parts := strings.Split(monthStr, "-")
		if len(parts) != 2 {
			return "", "", errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil)
		}
		y, _ = strconv.Atoi(parts[0])
		m, _ = strconv.Atoi(parts[1])
	} else {
		now := time.Now()
		y = now.Year()
		m = int(now.Month())
	}
	start = fmt.Sprintf("%04d-%02d-01", y, m)
	end = time.Date(y, time.Month(m+1), 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	return start, end, nil
}

var goalsBudgetSetCmd = &cobra.Command{
	Use:   "set <goal-id>",
	Short: "Set a monthly budget amount for a goal",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "goals.budget.set", "failed to set goal budget amount", safety.TierMutation, func() (mutation, *errors.Error) {
			if monthStr == "" {
				return mutation{}, errors.New(errors.InvalidArguments, "--month is required (YYYY-MM)", errors.CatValidation, false, nil)
			}
			month := monthStr + "-01"
			return mutation{
				resourceID: id,
				planAfter:  map[string]any{"month": month, "amount": goalAmount, "apply_to_future": goalApplyFuture, "account": goalAccountID},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.SetGoalBudgetAmount(ctx, id, month, goalAmount, goalApplyFuture, goalAccountID); err != nil {
						return nil, err
					}
					return map[string]string{"status": "budget amount set"}, nil
				},
				human: func() { fmt.Printf("Successfully set budget amount for goal %s.\n", id) },
			}, nil
		})
	},
}

func init() {
	goalsBudgetsCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	goalsCmd.AddCommand(goalsListCmd)
	goalsCmd.AddCommand(goalsShowCmd)
	goalsCmd.AddCommand(goalsCreateCmd)
	goalsCmd.AddCommand(goalsUpdateCmd)
	goalsCmd.AddCommand(goalsDeleteCmd)
	goalsCmd.AddCommand(goalsArchiveCmd)
	goalsCmd.AddCommand(goalsRestoreCmd)
	goalsCmd.AddCommand(goalsPrioritiesCmd)
	goalsCmd.AddCommand(goalsLinkAccountCmd)
	goalsCmd.AddCommand(goalsUnlinkAccountCmd)
	goalsCmd.AddCommand(goalsEventsCmd)
	goalsCmd.AddCommand(goalsBudgetCmd)
	goalsCmd.AddCommand(goalsBudgetsCmd)
	RootCmd.AddCommand(goalsCmd)

	goalsCreateCmd.Flags().StringVar(&goalName, "name", "", "goal name")
	goalsCreateCmd.Flags().StringVar(&goalType, "type", "", "goal type")
	goalsCreateCmd.Flags().Float64Var(&goalTargetAmount, "target-amount", 0, "target amount")
	goalsCreateCmd.Flags().StringVar(&goalTargetDate, "target-date", "", "target date (YYYY-MM-DD)")
	goalsCreateCmd.Flags().Float64Var(&goalMonthly, "monthly", 0, "planned monthly contribution")
	goalsCreateCmd.Flags().BoolVar(&goalSinking, "sinking-fund", false, "mark as sinking fund")
	goalsCreateCmd.Flags().IntVar(&goalPriority, "priority", 0, "priority")
	goalsCreateCmd.MarkFlagRequired("name") //nolint:errcheck // flag registered above

	goalsUpdateCmd.Flags().StringVar(&goalName, "name", "", "new name")
	goalsUpdateCmd.Flags().StringVar(&goalType, "type", "", "new type")
	goalsUpdateCmd.Flags().Float64Var(&goalTargetAmount, "target-amount", 0, "new target amount")
	goalsUpdateCmd.Flags().StringVar(&goalTargetDate, "target-date", "", "new target date (YYYY-MM-DD)")
	goalsUpdateCmd.Flags().Float64Var(&goalMonthly, "monthly", 0, "new planned monthly contribution")
	goalsUpdateCmd.Flags().BoolVar(&goalSinking, "sinking-fund", false, "mark as sinking fund")
	goalsUpdateCmd.Flags().IntVar(&goalPriority, "priority", 0, "new priority")

	goalsPrioritiesCmd.Flags().StringSliceVar(&goalPriorityIDs, "id", nil, "goal IDs in priority order (repeatable, first is highest)")
	goalsPrioritiesCmd.MarkFlagRequired("id") //nolint:errcheck // flag registered above

	goalsLinkAccountCmd.Flags().StringVar(&goalAccountID, "account", "", "account ID")
	goalsLinkAccountCmd.Flags().Float64Var(&goalAmount, "amount", 0, "contribution amount")
	goalsLinkAccountCmd.Flags().BoolVar(&goalEntireBalance, "entire-balance", true, "use the entire account balance")
	goalsLinkAccountCmd.MarkFlagRequired("account") //nolint:errcheck // flag registered above

	goalsUnlinkAccountCmd.Flags().StringVar(&goalAccountID, "account", "", "account ID")
	goalsUnlinkAccountCmd.MarkFlagRequired("account") //nolint:errcheck // flag registered above

	goalsEventsContributeCmd.Flags().StringVar(&goalAccountID, "account", "", "account ID")
	goalsEventsContributeCmd.Flags().Float64Var(&goalAmount, "amount", 0, "contribution amount")
	goalsEventsContributeCmd.Flags().StringVar(&goalDate, "date", "", "date (YYYY-MM-DD)")
	goalsEventsContributeCmd.Flags().StringVar(&goalNotes, "notes", "", "notes")
	goalsEventsContributeCmd.MarkFlagRequired("account") //nolint:errcheck // flag registered above
	goalsEventsContributeCmd.MarkFlagRequired("amount")  //nolint:errcheck // flag registered above

	goalsEventsWithdrawCmd.Flags().StringVar(&goalAccountID, "account", "", "account ID")
	goalsEventsWithdrawCmd.Flags().Float64Var(&goalAmount, "amount", 0, "withdrawal amount")
	goalsEventsWithdrawCmd.Flags().StringVar(&goalDate, "date", "", "date (YYYY-MM-DD)")
	goalsEventsWithdrawCmd.Flags().StringVar(&goalNotes, "notes", "", "notes")
	goalsEventsWithdrawCmd.MarkFlagRequired("account") //nolint:errcheck // flag registered above
	goalsEventsWithdrawCmd.MarkFlagRequired("amount")  //nolint:errcheck // flag registered above

	goalsEventsUpdateCmd.Flags().StringVar(&goalDate, "date", "", "new date (YYYY-MM-DD)")
	goalsEventsUpdateCmd.Flags().StringVar(&goalNotes, "notes", "", "new notes")

	goalsEventsCmd.AddCommand(goalsEventsListCmd)
	goalsEventsCmd.AddCommand(goalsEventsContributeCmd)
	goalsEventsCmd.AddCommand(goalsEventsWithdrawCmd)
	goalsEventsCmd.AddCommand(goalsEventsUpdateCmd)
	goalsEventsCmd.AddCommand(goalsEventsDeleteCmd)

	goalsBudgetCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	goalsBudgetSetCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")
	goalsBudgetSetCmd.Flags().Float64Var(&goalAmount, "amount", 0, "planned amount")
	goalsBudgetSetCmd.Flags().BoolVar(&goalApplyFuture, "apply-to-future", false, "apply to future months")
	goalsBudgetSetCmd.Flags().StringVar(&goalAccountID, "account", "", "account ID")
	goalsBudgetSetCmd.MarkFlagRequired("month")  //nolint:errcheck // flag registered above
	goalsBudgetSetCmd.MarkFlagRequired("amount") //nolint:errcheck // flag registered above
	goalsBudgetCmd.AddCommand(goalsBudgetSetCmd)
}
