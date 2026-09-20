package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
)

var (
	recurringAmount    float64
	recurringFrequency string
	recurringBaseDate  string
	recurringIsActive  bool
	recurringMerchant  string
)

var recurringCmd = &cobra.Command{
	Use:     "recurring",
	Short:   "Manage recurring transactions",
	GroupID: "core",
	Example: "  monarch recurring list --json",
}

var recurringListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recurring transactions",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "recurring.list", "failed to list recurring transactions",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.RecurringTransaction, error) {
				now := time.Now()
				startDate := now.Format("2006-01-02")
				endDate := time.Date(now.Year(), now.Month()+2, 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
				return svc.ListRecurring(ctx, startDate, endDate)
			},
			func(recurring []monarch.RecurringTransaction) {
				fmt.Printf("%-20s %10s %-12s %-12s %s\n", "MERCHANT", "AMOUNT", "FREQUENCY", "NEXT DATE", "STATUS")
				for _, r := range recurring {
					fmt.Printf("%-20s %10.2f %-12s %-12s %s\n", r.Merchant, r.Amount, r.Frequency, r.NextDate, r.Status)
				}
			})
	},
}

var recurringUpdateCmd = &cobra.Command{
	Use:   "update <recurring-id>",
	Short: "Update a recurring transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "recurring.update", "failed to update recurring transaction", safety.TierMutation, func() (mutation, *errors.Error) {
			var r *monarch.RecurringTransaction
			return mutation{
				resourceID: id,
				planAfter:  map[string]any{"amount": recurringAmount},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateRecurring(ctx, id, recurringAmount)
					if err != nil {
						return nil, err
					}
					r = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Successfully updated recurring transaction %s.\n", r.ID) },
			}, nil
		})
	},
}

var recurringStreamsCmd = &cobra.Command{
	Use:   "streams",
	Short: "List recurring streams with forecast details",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "recurring.streams", "failed to list recurring streams",
			func(ctx context.Context, svc *monarch.Service) ([]*monarch.RecurringStreamDetail, error) {
				return svc.ListRecurringStreams(ctx)
			},
			func(streams []*monarch.RecurringStreamDetail) {
				fmt.Printf("%-20s %-20s %10s %-12s %s\n", "ID", "MERCHANT", "AMOUNT", "FREQUENCY", "NEXT")
				for _, s := range streams {
					fmt.Printf("%-20s %-20s %10.2f %-12s %s\n", s.ID, s.MerchantName, s.Amount, s.Frequency, s.NextDate)
				}
			})
	},
}

var recurringShowCmd = &cobra.Command{
	Use:   "show <stream-id>",
	Short: "Show a recurring stream",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "recurring.show", "failed to get recurring stream",
			func(ctx context.Context, svc *monarch.Service) (*monarch.RecurringStreamDetail, error) {
				return svc.GetRecurringStream(ctx, args[0])
			},
			func(s *monarch.RecurringStreamDetail) {
				fmt.Printf("ID:        %s\n", s.ID)
				fmt.Printf("Merchant:  %s\n", s.MerchantName)
				fmt.Printf("Amount:    %.2f\n", s.Amount)
				fmt.Printf("Frequency: %s\n", s.Frequency)
				fmt.Printf("Next:      %s %.2f\n", s.NextDate, s.NextAmount)
			})
	},
}

var recurringSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Summarize upcoming recurring income and expenses",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "recurring.summary", "failed to get recurring summary",
			func(ctx context.Context, svc *monarch.Service) (*monarch.RecurringSummary, error) {
				now := time.Now()
				startDate := now.Format("2006-01-02")
				endDate := time.Date(now.Year(), now.Month()+2, 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
				return svc.GetRecurringSummary(ctx, startDate, endDate)
			},
			func(summary *monarch.RecurringSummary) {
				fmt.Printf("Expense: %.2f total (%d items, %.2f completed, %.2f remaining)\n",
					summary.ExpenseTotal, summary.ExpenseCount, summary.ExpenseCompleted, summary.ExpenseRemaining)
				fmt.Printf("Income:  %.2f total (%.2f completed, %.2f remaining)\n",
					summary.IncomeTotal, summary.IncomeCompleted, summary.IncomeRemaining)
			})
	},
}

var recurringCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a recurring stream for a merchant",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "recurring.create", "failed to create recurring stream", safety.TierMutation, func() (mutation, *errors.Error) {
			if recurringMerchant == "" {
				return mutation{}, errors.New(errors.InvalidArguments, "--merchant is required", errors.CatValidation, false, nil)
			}
			input := &monarch.RecurringStreamInput{}
			after := map[string]any{"merchant": recurringMerchant}
			if cmd.Flags().Changed("frequency") {
				input.Frequency = &recurringFrequency
				after["frequency"] = recurringFrequency
			}
			if cmd.Flags().Changed("amount") {
				input.Amount = &recurringAmount
				after["amount"] = recurringAmount
			}
			if cmd.Flags().Changed("date") {
				input.BaseDate = &recurringBaseDate
				after["date"] = recurringBaseDate
			}
			if cmd.Flags().Changed("active") {
				input.IsActive = &recurringIsActive
				after["active"] = recurringIsActive
			}
			var stream *monarch.RecurringStreamDetail
			return mutation{
				planAfter: after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					created, err := svc.CreateRecurringStream(ctx, recurringMerchant, input)
					if err != nil {
						return nil, err
					}
					stream = created
					return created, nil
				},
				human: func() { fmt.Printf("Successfully created recurring stream %s.\n", stream.ID) },
			}, nil
		})
	},
}

var recurringStreamUpdateCmd = &cobra.Command{
	Use:   "stream-update <stream-id>",
	Short: "Update a recurring stream (frequency, amount, date, active)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "recurring.stream-update", "failed to update recurring stream", safety.TierMutation, func() (mutation, *errors.Error) {
			input := &monarch.RecurringStreamInput{}
			after := map[string]any{}
			if cmd.Flags().Changed("frequency") {
				input.Frequency = &recurringFrequency
				after["frequency"] = recurringFrequency
			}
			if cmd.Flags().Changed("amount") {
				input.Amount = &recurringAmount
				after["amount"] = recurringAmount
			}
			if cmd.Flags().Changed("date") {
				input.BaseDate = &recurringBaseDate
				after["date"] = recurringBaseDate
			}
			if cmd.Flags().Changed("active") {
				input.IsActive = &recurringIsActive
				after["active"] = recurringIsActive
			}
			if len(after) == 0 {
				return mutation{}, errors.New(errors.InvalidArguments, "at least one of --frequency, --amount, --date, --active is required", errors.CatValidation, false, nil)
			}
			var stream *monarch.RecurringStreamDetail
			return mutation{
				resourceID: id,
				planAfter:  after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateRecurringStream(ctx, id, input)
					if err != nil {
						return nil, err
					}
					stream = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Successfully updated recurring stream %s.\n", stream.ID) },
			}, nil
		})
	},
}

var recurringRemoveCmd = &cobra.Command{
	Use:   "remove <stream-id>",
	Short: "Mark a stream as not recurring",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "recurring.remove", "failed to remove recurring stream", safety.TierDestructive, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.RemoveRecurringStream(ctx, id); err != nil {
						return nil, err
					}
					return map[string]string{"status": "removed"}, nil
				},
				human: func() { fmt.Printf("Successfully removed recurring stream %s.\n", id) },
			}, nil
		})
	},
}

func init() {
	recurringUpdateCmd.Flags().Float64Var(&recurringAmount, "amount", 0, "new recurring amount")
	recurringUpdateCmd.MarkFlagRequired("amount") //nolint:errcheck // flag registered above

	recurringCreateCmd.Flags().StringVar(&recurringMerchant, "merchant", "", "merchant ID")
	recurringCreateCmd.Flags().StringVar(&recurringFrequency, "frequency", "", "frequency (e.g. monthly)")
	recurringCreateCmd.Flags().Float64Var(&recurringAmount, "amount", 0, "expected amount")
	recurringCreateCmd.Flags().StringVar(&recurringBaseDate, "date", "", "base date (YYYY-MM-DD)")
	recurringCreateCmd.Flags().BoolVar(&recurringIsActive, "active", true, "whether the stream is active")
	recurringCreateCmd.MarkFlagRequired("merchant") //nolint:errcheck // flag registered above

	recurringStreamUpdateCmd.Flags().StringVar(&recurringFrequency, "frequency", "", "new frequency")
	recurringStreamUpdateCmd.Flags().Float64Var(&recurringAmount, "amount", 0, "new amount")
	recurringStreamUpdateCmd.Flags().StringVar(&recurringBaseDate, "date", "", "new base date (YYYY-MM-DD)")
	recurringStreamUpdateCmd.Flags().BoolVar(&recurringIsActive, "active", true, "whether the stream is active")

	recurringCmd.AddCommand(recurringListCmd)
	recurringCmd.AddCommand(recurringStreamsCmd)
	recurringCmd.AddCommand(recurringShowCmd)
	recurringCmd.AddCommand(recurringSummaryCmd)
	recurringCmd.AddCommand(recurringUpdateCmd)
	recurringCmd.AddCommand(recurringCreateCmd)
	recurringCmd.AddCommand(recurringStreamUpdateCmd)
	recurringCmd.AddCommand(recurringRemoveCmd)
	RootCmd.AddCommand(recurringCmd)
}
