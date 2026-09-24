package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
)

var (
	ruleMerchantOperator                    string
	ruleMerchantValue                       string
	ruleLegacyMerchantOperator              string
	ruleLegacyMerchantValue                 string
	ruleUseOriginalStatement                bool
	ruleOriginalStatementOperator           string
	ruleOriginalStatementValue              string
	ruleAmountOperator                      string
	ruleAmountValue                         float64
	ruleAmountValueUpper                    float64
	ruleAmountIsExpense                     bool
	ruleCategoryIDs                         []string
	ruleSetCategoryID                       string
	ruleSetMerchant                         string
	ruleAddTagIDs                           []string
	ruleAccountIDs                          []string
	ruleCriteriaOwnerUserIDs                []string
	ruleCriteriaOwnerIsJoint                bool
	ruleCriteriaBusinessEntityIDs           []string
	ruleCriteriaBusinessEntityIsUnassigned  bool
	ruleHideFromReports                     bool
	ruleReviewStatus                        string
	ruleNeedsReviewByUserID                 string
	ruleLinkGoalID                          string
	ruleLinkSavingsGoalID                   string
	ruleLinkToPaydownBudget                 bool
	ruleSendNotification                    bool
	ruleActionSetOwner                      string
	ruleActionSetOwnerIsJoint               bool
	ruleActionSetBusinessEntity             string
	ruleActionSetBusinessEntityIsUnassigned bool
	ruleSplitFile                           string
	ruleApplyToExisting                     bool
	ruleOrder                               int
)

var rulesCmd = &cobra.Command{
	Use:     "rules",
	Short:   "Manage transaction auto-categorization rules",
	GroupID: "core",
	Example: "  monarch rules list --json\n  monarch rules create --merchant-operator contains --merchant-value \"Uber\" --set-category-id <id> --confirm",
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
					criteria := r.MerchantNameCriteria
					if len(criteria) == 0 {
						criteria = r.OriginalStatementCriteria
					}
					if len(criteria) == 0 {
						criteria = r.MerchantCriteria
					}
					match := ""
					operator := ""
					if len(criteria) > 0 {
						match = criteria[0].Value
						operator = criteria[0].Operator
					}
					var actions []string
					if r.SetCategoryAction != nil {
						actions = append(actions, "→ "+r.SetCategoryAction.Name)
					}
					if r.SetMerchantAction != nil {
						actions = append(actions, "rename to "+r.SetMerchantAction.Name)
					}
					if len(r.AddTagsAction) > 0 {
						names := make([]string, len(r.AddTagsAction))
						for i, t := range r.AddTagsAction {
							names[i] = t.Name
						}
						actions = append(actions, "tag "+strings.Join(names, ","))
					}
					if r.ReviewStatusAction != nil {
						actions = append(actions, "review:"+*r.ReviewStatusAction)
					}
					if r.SetHideFromReportsAction != nil && *r.SetHideFromReportsAction {
						actions = append(actions, "hide")
					}
					fmt.Printf("%-36s %-12s %-20s %s\n", r.ID, operator, match, strings.Join(actions, ", "))
				}
				fmt.Printf("\nTotal rules: %d\n", len(rules))
			})
	},
}

func buildRuleFields(cmd *cobra.Command) (monarch.RuleFields, *errors.Error) {
	fields := monarch.RuleFields{
		MerchantOperator:          ruleMerchantOperator,
		MerchantValue:             ruleMerchantValue,
		LegacyMerchantOperator:    ruleLegacyMerchantOperator,
		LegacyMerchantValue:       ruleLegacyMerchantValue,
		OriginalStatementOperator: ruleOriginalStatementOperator,
		OriginalStatementValue:    ruleOriginalStatementValue,
		AmountOperator:            ruleAmountOperator,
		AmountIsExpense:           ruleAmountIsExpense,
		CategoryIDs:               ruleCategoryIDs,
		SetCategoryID:             ruleSetCategoryID,
		SetMerchant:               ruleSetMerchant,
		AddTagIDs:                 ruleAddTagIDs,
		AccountIDs:                ruleAccountIDs,
		CriteriaOwnerUserIDs:      ruleCriteriaOwnerUserIDs,
		CriteriaBusinessEntityIDs: ruleCriteriaBusinessEntityIDs,
		ReviewStatus:              ruleReviewStatus,
		NeedsReviewByUserID:       ruleNeedsReviewByUserID,
		LinkGoalID:                ruleLinkGoalID,
		LinkSavingsGoalID:         ruleLinkSavingsGoalID,
		ActionSetOwner:            ruleActionSetOwner,
		ActionSetBusinessEntity:   ruleActionSetBusinessEntity,
		ApplyToExisting:           ruleApplyToExisting,
	}
	if cmd.Flags().Changed("amount-value") {
		fields.AmountValue = &ruleAmountValue
	}
	if cmd.Flags().Changed("amount-value-upper") {
		fields.AmountValueUpper = &ruleAmountValueUpper
	}
	if cmd.Flags().Changed("use-original-statement") {
		fields.UseOriginalStatement = &ruleUseOriginalStatement
	}
	if cmd.Flags().Changed("criteria-owner-joint") {
		fields.CriteriaOwnerIsJoint = &ruleCriteriaOwnerIsJoint
	}
	if cmd.Flags().Changed("criteria-business-entity-unassigned") {
		fields.CriteriaBusinessEntityIsUnassigned = &ruleCriteriaBusinessEntityIsUnassigned
	}
	if cmd.Flags().Changed("hide-from-reports") {
		fields.HideFromReports = &ruleHideFromReports
	}
	if cmd.Flags().Changed("link-to-paydown-budget") {
		fields.LinkToPaydownBudget = &ruleLinkToPaydownBudget
	}
	if cmd.Flags().Changed("send-notification") {
		fields.SendNotification = &ruleSendNotification
	}
	if cmd.Flags().Changed("action-set-owner-joint") {
		fields.ActionSetOwnerIsJoint = &ruleActionSetOwnerIsJoint
	}
	if cmd.Flags().Changed("action-set-business-entity-unassigned") {
		fields.ActionSetBusinessEntityIsUnassigned = &ruleActionSetBusinessEntityIsUnassigned
	}
	if fields.ReviewStatus != "" && fields.ReviewStatus != "needs_review" && fields.ReviewStatus != "reviewed" {
		return monarch.RuleFields{}, errors.New(errors.InvalidArguments, "--review-status must be needs_review or reviewed", errors.CatValidation, false, nil)
	}
	if fields.AmountOperator == "between" && fields.AmountValueUpper == nil {
		return monarch.RuleFields{}, errors.New(errors.InvalidArguments, "--amount-value-upper is required when --amount-operator=between", errors.CatValidation, false, nil)
	}
	if fields.AmountValueUpper != nil && fields.AmountOperator != "between" {
		return monarch.RuleFields{}, errors.New(errors.InvalidArguments, "--amount-value-upper requires --amount-operator=between", errors.CatValidation, false, nil)
	}
	if ruleSplitFile != "" {
		data, err := os.ReadFile(ruleSplitFile)
		if err != nil {
			return monarch.RuleFields{}, errors.New(errors.InvalidArguments, "failed to read split file: "+err.Error(), errors.CatValidation, false, err)
		}
		var split monarch.RuleSplitAction
		if err := json.Unmarshal(data, &split); err != nil {
			return monarch.RuleFields{}, errors.New(errors.InvalidArguments, "invalid split JSON: "+err.Error(), errors.CatValidation, false, err)
		}
		if split.AmountType != "ABSOLUTE" && split.AmountType != "PERCENTAGE" {
			return monarch.RuleFields{}, errors.New(errors.InvalidArguments, "split amountType must be ABSOLUTE or PERCENTAGE", errors.CatValidation, false, nil)
		}
		if len(split.SplitsInfo) == 0 {
			return monarch.RuleFields{}, errors.New(errors.InvalidArguments, "split splitsInfo must not be empty", errors.CatValidation, false, nil)
		}
		fields.SplitAction = &split
	}
	return fields, nil
}

var rulesCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a transaction rule (requires --confirm)",
	Long:    `Create an auto-categorization rule. Match by merchant (--merchant-operator eq|contains with --merchant-value), raw bank statement text (--original-statement-operator eq|contains with --original-statement-value), amount (--amount-operator gt|lt|eq|between with --amount-value, plus --amount-value-upper for between), categories (--category-id), accounts (--account-id), owners, or business entities. Then apply updates: category (--set-category-id), merchant rename (--set-merchant), tags (--add-tag-id), review status (--review-status needs_review|reviewed), hide (--hide-from-reports), goal links, owner, business, notifications, or splits (--split-file JSON with amountType ABSOLUTE|PERCENTAGE and splitsInfo). Rules evaluate in list order: check rules list and use rules reorder to prioritize. --apply-to-existing retro-applies to current transactions.`,
	Example: `  monarch rules create --merchant-operator contains --merchant-value "Uber" --set-category-id <id> --dry-run --json`,
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "rules.create", "failed to create rule", safety.TierMutation, func() (mutation, *errors.Error) {
			fields, verr := buildRuleFields(cmd)
			if verr != nil {
				return mutation{}, verr
			}
			input := monarch.CreateRuleInput{RuleFields: fields}
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
	Short: "Update a transaction rule (requires --confirm)",
	Long:  `Update an auto-categorization rule; accepts the same match criteria and actions as rules create. Only the flags passed are sent.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "rules.update", "failed to update rule", safety.TierMutation, func() (mutation, *errors.Error) {
			fields, verr := buildRuleFields(cmd)
			if verr != nil {
				return mutation{}, verr
			}
			input := monarch.UpdateRuleInput{ID: id, RuleFields: fields}
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
	Use:     "reorder <rule-id>",
	Short:   "Move a rule to a new position in the evaluation order (requires --confirm)",
	Long:    `Move a rule to a zero-based position; earlier rules match first. See the current order with rules list.`,
	Example: `  monarch rules reorder <rule-id> --order 0 --dry-run --json`,
	Args:    cobra.ExactArgs(1),
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
	Short: "Delete a transaction rule (requires --confirm)",
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

func registerRuleFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&ruleMerchantOperator, "merchant-operator", "", "merchant match operator (eq, contains)")
	cmd.Flags().StringVar(&ruleMerchantValue, "merchant-value", "", "merchant name/pattern to match")
	cmd.Flags().StringVar(&ruleLegacyMerchantOperator, "merchant-criteria-operator", "", "legacy merchant match operator (eq, contains)")
	cmd.Flags().StringVar(&ruleLegacyMerchantValue, "merchant-criteria-value", "", "legacy merchant name/pattern to match")
	cmd.Flags().BoolVar(&ruleUseOriginalStatement, "use-original-statement", false, "match legacy merchant criteria against the raw bank statement text")
	cmd.Flags().StringVar(&ruleOriginalStatementOperator, "original-statement-operator", "", "raw statement match operator (eq, contains)")
	cmd.Flags().StringVar(&ruleOriginalStatementValue, "original-statement-value", "", "raw bank statement text/pattern to match")
	cmd.Flags().StringVar(&ruleAmountOperator, "amount-operator", "", "amount comparison (gt, lt, eq, between)")
	cmd.Flags().Float64Var(&ruleAmountValue, "amount-value", 0, "amount threshold value")
	cmd.Flags().Float64Var(&ruleAmountValueUpper, "amount-value-upper", 0, "upper bound for between comparisons")
	cmd.Flags().BoolVar(&ruleAmountIsExpense, "amount-is-expense", true, "whether amount is expense")
	cmd.Flags().StringSliceVar(&ruleCategoryIDs, "category-id", nil, "limit rule to category IDs (repeatable)")
	cmd.Flags().StringVar(&ruleSetCategoryID, "set-category-id", "", "category ID to assign")
	cmd.Flags().StringVar(&ruleSetMerchant, "set-merchant", "", "merchant name to rename transactions to")
	cmd.Flags().StringSliceVar(&ruleAddTagIDs, "add-tag-id", nil, "tag IDs to add (repeatable)")
	cmd.Flags().StringSliceVar(&ruleAccountIDs, "account-id", nil, "limit rule to account IDs (repeatable)")
	cmd.Flags().StringSliceVar(&ruleCriteriaOwnerUserIDs, "criteria-owner-user-id", nil, "limit rule to owner user IDs (repeatable)")
	cmd.Flags().BoolVar(&ruleCriteriaOwnerIsJoint, "criteria-owner-joint", false, "limit rule to jointly owned transactions")
	cmd.Flags().StringSliceVar(&ruleCriteriaBusinessEntityIDs, "criteria-business-entity-id", nil, "limit rule to business entity IDs (repeatable)")
	cmd.Flags().BoolVar(&ruleCriteriaBusinessEntityIsUnassigned, "criteria-business-entity-unassigned", false, "limit rule to transactions without a business entity")
	cmd.Flags().BoolVar(&ruleHideFromReports, "hide-from-reports", false, "hide matching transactions from reports")
	cmd.Flags().StringVar(&ruleReviewStatus, "review-status", "", "review status to apply (needs_review, reviewed)")
	cmd.Flags().StringVar(&ruleNeedsReviewByUserID, "needs-review-by-user-id", "", "assign review to a household user ID")
	cmd.Flags().StringVar(&ruleLinkGoalID, "link-goal-id", "", "goal ID to link transactions to")
	cmd.Flags().StringVar(&ruleLinkSavingsGoalID, "link-savings-goal-id", "", "savings goal ID to link transactions to")
	cmd.Flags().BoolVar(&ruleLinkToPaydownBudget, "link-to-paydown-budget", false, "link matching transactions to pay down tracking")
	cmd.Flags().BoolVar(&ruleSendNotification, "send-notification", false, "send a notification for matching transactions")
	cmd.Flags().StringVar(&ruleActionSetOwner, "action-set-owner", "", "assign ownership to a household user ID")
	cmd.Flags().BoolVar(&ruleActionSetOwnerIsJoint, "action-set-owner-joint", false, "assign ownership jointly")
	cmd.Flags().StringVar(&ruleActionSetBusinessEntity, "action-set-business-entity", "", "assign a business entity ID")
	cmd.Flags().BoolVar(&ruleActionSetBusinessEntityIsUnassigned, "action-set-business-entity-unassigned", false, "clear the business entity assignment")
	cmd.Flags().StringVar(&ruleSplitFile, "split-file", "", "JSON file with amountType (ABSOLUTE|PERCENTAGE) and splitsInfo for auto-split")
	cmd.Flags().BoolVar(&ruleApplyToExisting, "apply-to-existing", false, "apply rule to existing transactions")
}

func init() {
	registerRuleFlags(rulesCreateCmd)
	registerRuleFlags(rulesUpdateCmd)

	rulesReorderCmd.Flags().IntVar(&ruleOrder, "order", 0, "zero-based target position (0 runs first)")
	rulesReorderCmd.MarkFlagRequired("order") //nolint:errcheck // flag registered above

	rulesCmd.AddCommand(rulesListCmd)
	rulesCmd.AddCommand(rulesCreateCmd)
	rulesCmd.AddCommand(rulesUpdateCmd)
	rulesCmd.AddCommand(rulesReorderCmd)
	rulesCmd.AddCommand(rulesDeleteCmd)
	RootCmd.AddCommand(rulesCmd)
}
