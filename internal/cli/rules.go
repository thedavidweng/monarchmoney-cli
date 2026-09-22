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
	ruleMerchantOperator string
	ruleMerchantValue    string
	ruleAmountOperator   string
	ruleAmountValue      float64
	ruleAmountIsExpense  bool
	ruleSetCategoryID    string
	ruleAddTagIDs        []string
	ruleAccountIDs       []string
	ruleApplyToExisting  bool
	ruleOrder            int
)

var rulesCmd = &cobra.Command{
	Use:     "rules",
	Short:   "Manage transaction auto-categorization rules",
	GroupID: "core",
	Example: "  monarch rules list --json\n  monarch rules create --trigger-value \"Uber\" --category-id <id> --confirm",
}

var rulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all transaction rules",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "rules.list", "failed to list rules",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.Rule, error) {
				return svc.ListRules(ctx)
			},
			func(rules []monarch.Rule) {
				fmt.Printf("%-36s %-12s %-20s %s\n", "ID", "OPERATOR", "MATCH", "ACTION")
				for _, r := range rules {
					match := ""
					if len(r.MerchantNameCriteria) > 0 {
						match = r.MerchantNameCriteria[0].Value
					}
					action := ""
					if r.SetCategoryAction != nil {
						action = "→ " + r.SetCategoryAction.Name
					}
					operator := ""
					if len(r.MerchantNameCriteria) > 0 {
						operator = r.MerchantNameCriteria[0].Operator
					}
					fmt.Printf("%-36s %-12s %-20s %s\n", r.ID, operator, match, action)
				}
				fmt.Printf("\nTotal rules: %d\n", len(rules))
			})
	},
}

var rulesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a transaction rule",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "rules.create", "failed to create rule", safety.TierMutation, func() (mutation, *errors.Error) {
			input := monarch.CreateRuleInput{
				MerchantOperator: ruleMerchantOperator,
				MerchantValue:    ruleMerchantValue,
				AmountOperator:   ruleAmountOperator,
				AmountIsExpense:  ruleAmountIsExpense,
				SetCategoryID:    ruleSetCategoryID,
				AddTagIDs:        ruleAddTagIDs,
				AccountIDs:       ruleAccountIDs,
				ApplyToExisting:  ruleApplyToExisting,
			}
			if cmd.Flags().Changed("amount-value") {
				input.AmountValue = &ruleAmountValue
			}
			return mutation{
				planAfter: map[string]any{"input": input},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.CreateRule(ctx, &input); err != nil {
						return nil, err
					}
					return map[string]string{"status": "created"}, nil
				},
				human: func() { fmt.Println("Successfully created rule.") },
			}, nil
		})
	},
}

var rulesUpdateCmd = &cobra.Command{
	Use:   "update <rule-id>",
	Short: "Update a transaction rule",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "rules.update", "failed to update rule", safety.TierMutation, func() (mutation, *errors.Error) {
			input := monarch.UpdateRuleInput{
				ID:               id,
				MerchantOperator: ruleMerchantOperator,
				MerchantValue:    ruleMerchantValue,
				AmountOperator:   ruleAmountOperator,
				AmountIsExpense:  ruleAmountIsExpense,
				SetCategoryID:    ruleSetCategoryID,
				AddTagIDs:        ruleAddTagIDs,
				AccountIDs:       ruleAccountIDs,
				ApplyToExisting:  ruleApplyToExisting,
			}
			if cmd.Flags().Changed("amount-value") {
				input.AmountValue = &ruleAmountValue
			}
			return mutation{
				resourceID: id,
				planAfter:  map[string]any{"input": input},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.UpdateRule(ctx, &input); err != nil {
						return nil, err
					}
					return map[string]string{"status": "updated"}, nil
				},
				human: func() { fmt.Printf("Successfully updated rule %s.\n", id) },
			}, nil
		})
	},
}

var rulesReorderCmd = &cobra.Command{
	Use:   "reorder <rule-id>",
	Short: "Move a rule to a new position in the evaluation order",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "rules.reorder", "failed to reorder rule", safety.TierMutation, func() (mutation, *errors.Error) {
			if ruleOrder < 0 {
				return mutation{}, errors.New(errors.InvalidArguments, "--order must be 0 or greater", errors.CatValidation, false, nil)
			}
			var result *monarch.RuleReorderResult
			return mutation{
				resourceID: id,
				planAfter:  map[string]any{"order": ruleOrder},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					reordered, err := svc.ReorderRule(ctx, id, ruleOrder)
					if err != nil {
						return nil, err
					}
					result = reordered
					return reordered, nil
				},
				human: func() { fmt.Printf("Moved rule %s from %d to %d.\n", id, result.MovedFrom, result.MovedTo) },
			}, nil
		})
	},
}

var rulesDeleteCmd = &cobra.Command{
	Use:   "delete <rule-id>",
	Short: "Delete a transaction rule",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "rules.delete", "failed to delete rule", safety.TierDestructive, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.DeleteRule(ctx, id); err != nil {
						return nil, err
					}
					return map[string]string{"status": "deleted"}, nil
				},
				human: func() { fmt.Printf("Successfully deleted rule %s.\n", id) },
			}, nil
		})
	},
}

func init() {
	rulesCreateCmd.Flags().StringVar(&ruleMerchantOperator, "merchant-operator", "", "merchant match operator (eq, contains)")
	rulesCreateCmd.Flags().StringVar(&ruleMerchantValue, "merchant-value", "", "merchant name/pattern to match")
	rulesCreateCmd.Flags().StringVar(&ruleAmountOperator, "amount-operator", "", "amount comparison (gt, lt, eq, between)")
	rulesCreateCmd.Flags().Float64Var(&ruleAmountValue, "amount-value", 0, "amount threshold value")
	rulesCreateCmd.Flags().BoolVar(&ruleAmountIsExpense, "amount-is-expense", true, "whether amount is expense")
	rulesCreateCmd.Flags().StringVar(&ruleSetCategoryID, "set-category-id", "", "category ID to assign")
	rulesCreateCmd.Flags().StringSliceVar(&ruleAddTagIDs, "add-tag-id", nil, "tag IDs to add (repeatable)")
	rulesCreateCmd.Flags().StringSliceVar(&ruleAccountIDs, "account-id", nil, "limit rule to account IDs (repeatable)")
	rulesCreateCmd.Flags().BoolVar(&ruleApplyToExisting, "apply-to-existing", false, "apply rule to existing transactions")

	rulesUpdateCmd.Flags().StringVar(&ruleMerchantOperator, "merchant-operator", "", "merchant match operator (eq, contains)")
	rulesUpdateCmd.Flags().StringVar(&ruleMerchantValue, "merchant-value", "", "merchant name/pattern to match")
	rulesUpdateCmd.Flags().StringVar(&ruleAmountOperator, "amount-operator", "", "amount comparison (gt, lt, eq, between)")
	rulesUpdateCmd.Flags().Float64Var(&ruleAmountValue, "amount-value", 0, "amount threshold value")
	rulesUpdateCmd.Flags().BoolVar(&ruleAmountIsExpense, "amount-is-expense", true, "whether amount is expense")
	rulesUpdateCmd.Flags().StringVar(&ruleSetCategoryID, "set-category-id", "", "category ID to assign")
	rulesUpdateCmd.Flags().StringSliceVar(&ruleAddTagIDs, "add-tag-id", nil, "tag IDs to add (repeatable)")
	rulesUpdateCmd.Flags().StringSliceVar(&ruleAccountIDs, "account-id", nil, "limit rule to account IDs (repeatable)")
	rulesUpdateCmd.Flags().BoolVar(&ruleApplyToExisting, "apply-to-existing", false, "apply rule to existing transactions")

	rulesReorderCmd.Flags().IntVar(&ruleOrder, "order", 0, "zero-based target position (0 runs first)")
	rulesReorderCmd.MarkFlagRequired("order") //nolint:errcheck // flag registered above

	rulesCmd.AddCommand(rulesListCmd)
	rulesCmd.AddCommand(rulesCreateCmd)
	rulesCmd.AddCommand(rulesUpdateCmd)
	rulesCmd.AddCommand(rulesReorderCmd)
	rulesCmd.AddCommand(rulesDeleteCmd)
	RootCmd.AddCommand(rulesCmd)
}
