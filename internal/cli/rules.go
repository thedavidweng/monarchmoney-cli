package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
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
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "rules.list", start)
		if !ok {
			return
		}
		svc := deps.Service

		rules, err := svc.ListRules(cmd.Context())
		if err != nil {
			handleError(renderer, "rules.list", wrapError(err, "failed to list rules"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("rules.list", profile, output.SchemaVersion, "", rules, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
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
		}
	},
}

var rulesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a transaction rule",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "rules.create", err.(*errors.Error), start)
			return
		}

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

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("rules.create", "", nil, map[string]interface{}{"input": input})
			env := output.NewEnvelope("rules.create", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		deps, ok := newDeps(renderer, "rules.create", start)
		if !ok {
			return
		}

		if _, err := deps.Mutate("rules.create", "", func() (interface{}, error) {
			return nil, deps.Service.CreateRule(cmd.Context(), input)
		}, "failed to create rule"); err != nil {
			return
		}

		if jsonMode {
			env := output.NewEnvelope("rules.create", profile, output.SchemaVersion, "", map[string]string{"status": "created"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Println("Successfully created rule.")
		}
	},
}

var rulesUpdateCmd = &cobra.Command{
	Use:   "update <rule-id>",
	Short: "Update a transaction rule",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		id := args[0]

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "rules.update", err.(*errors.Error), start)
			return
		}

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

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("rules.update", id, nil, map[string]interface{}{"input": input})
			env := output.NewEnvelope("rules.update", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		deps, ok := newDeps(renderer, "rules.update", start)
		if !ok {
			return
		}

		if _, err := deps.Mutate("rules.update", id, func() (interface{}, error) {
			return nil, deps.Service.UpdateRule(cmd.Context(), input)
		}, "failed to update rule"); err != nil {
			return
		}

		if jsonMode {
			env := output.NewEnvelope("rules.update", profile, output.SchemaVersion, "", map[string]string{"status": "updated"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully updated rule %s.\n", id)
		}
	},
}

var rulesDeleteCmd = &cobra.Command{
	Use:   "delete <rule-id>",
	Short: "Delete a transaction rule",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		id := args[0]

		if err := safety.Check(safety.TierDestructive, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "rules.delete", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("rules.delete", id, nil, nil)
			env := output.NewEnvelope("rules.delete", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		deps, ok := newDeps(renderer, "rules.delete", start)
		if !ok {
			return
		}

		if _, err := deps.Mutate("rules.delete", id, func() (interface{}, error) {
			return nil, deps.Service.DeleteRule(cmd.Context(), id)
		}, "failed to delete rule"); err != nil {
			return
		}

		if jsonMode {
			env := output.NewEnvelope("rules.delete", profile, output.SchemaVersion, "", map[string]string{"status": "deleted"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully deleted rule %s.\n", id)
		}
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

	rulesCmd.AddCommand(rulesListCmd)
	rulesCmd.AddCommand(rulesCreateCmd)
	rulesCmd.AddCommand(rulesUpdateCmd)
	rulesCmd.AddCommand(rulesDeleteCmd)
	RootCmd.AddCommand(rulesCmd)
}
