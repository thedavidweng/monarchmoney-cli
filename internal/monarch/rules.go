package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetTransactionRulesQuery = queries.Get("rules/list.graphql")
var CreateTransactionRuleMutation = queries.Get("rules/create.graphql")
var UpdateTransactionRuleMutation = queries.Get("rules/update.graphql")
var DeleteTransactionRuleMutation = queries.Get("rules/delete.graphql")

type RuleCriteria struct {
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type RuleAmountCriteria struct {
	Operator  string  `json:"operator"`
	IsExpense bool    `json:"is_expense"`
	Value     float64 `json:"value"`
}

type RuleAction struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RuleTagAction struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Rule struct {
	ID                       string              `json:"id"`
	Order                    int                 `json:"order"`
	MerchantNameCriteria     []RuleCriteria      `json:"merchant_name_criteria,omitempty"`
	AmountCriteria           *RuleAmountCriteria `json:"amount_criteria,omitempty"`
	CategoryIDs              []string            `json:"category_ids,omitempty"`
	AccountIDs               []string            `json:"account_ids,omitempty"`
	SetCategoryAction        *RuleAction         `json:"set_category_action,omitempty"`
	SetMerchantAction        *RuleAction         `json:"set_merchant_action,omitempty"`
	AddTagsAction            []RuleTagAction     `json:"add_tags_action,omitempty"`
	SetHideFromReportsAction *bool               `json:"set_hide_from_reports_action,omitempty"`
	ReviewStatusAction       *string             `json:"review_status_action,omitempty"`
	RecentApplicationCount   int                 `json:"recent_application_count"`
	LastAppliedAt            string              `json:"last_applied_at,omitempty"`
}

type CreateRuleInput struct {
	MerchantOperator string
	MerchantValue    string
	AmountOperator   string
	AmountValue      *float64
	AmountIsExpense  bool
	SetCategoryID    string
	AddTagIDs        []string
	AccountIDs       []string
	ApplyToExisting  bool
}

type UpdateRuleInput struct {
	ID               string
	MerchantOperator string
	MerchantValue    string
	AmountOperator   string
	AmountValue      *float64
	AmountIsExpense  bool
	SetCategoryID    string
	AddTagIDs        []string
	AccountIDs       []string
	ApplyToExisting  bool
}

func (s *Service) ListRules(ctx context.Context) ([]Rule, error) {
	var resp struct {
		TransactionRules []struct {
			ID                   string `json:"id"`
			Order                int    `json:"order"`
			MerchantNameCriteria []struct {
				Operator string `json:"operator"`
				Value    string `json:"value"`
			} `json:"merchantNameCriteria"`
			AmountCriteria *struct {
				Operator  string  `json:"operator"`
				IsExpense bool    `json:"isExpense"`
				Value     float64 `json:"value"`
			} `json:"amountCriteria"`
			CategoryIDs       []string `json:"categoryIds"`
			AccountIDs        []string `json:"accountIds"`
			SetCategoryAction *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"setCategoryAction"`
			SetMerchantAction *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"setMerchantAction"`
			AddTagsAction []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"addTagsAction"`
			SetHideFromReportsAction *bool   `json:"setHideFromReportsAction"`
			ReviewStatusAction       *string `json:"reviewStatusAction"`
			RecentApplicationCount   int     `json:"recentApplicationCount"`
			LastAppliedAt            string  `json:"lastAppliedAt"`
		} `json:"transactionRules"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetTransactionRules",
		Query:         GetTransactionRulesQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}

	rules := make([]Rule, len(resp.TransactionRules))
	for i, r := range resp.TransactionRules {
		rule := Rule{
			ID:                       r.ID,
			Order:                    r.Order,
			CategoryIDs:              r.CategoryIDs,
			AccountIDs:               r.AccountIDs,
			SetHideFromReportsAction: r.SetHideFromReportsAction,
			ReviewStatusAction:       r.ReviewStatusAction,
			RecentApplicationCount:   r.RecentApplicationCount,
			LastAppliedAt:            r.LastAppliedAt,
		}
		for _, mc := range r.MerchantNameCriteria {
			rule.MerchantNameCriteria = append(rule.MerchantNameCriteria, RuleCriteria{Operator: mc.Operator, Value: mc.Value})
		}
		if r.AmountCriteria != nil {
			rule.AmountCriteria = &RuleAmountCriteria{Operator: r.AmountCriteria.Operator, IsExpense: r.AmountCriteria.IsExpense, Value: r.AmountCriteria.Value}
		}
		if r.SetCategoryAction != nil {
			rule.SetCategoryAction = &RuleAction{ID: r.SetCategoryAction.ID, Name: r.SetCategoryAction.Name}
		}
		if r.SetMerchantAction != nil {
			rule.SetMerchantAction = &RuleAction{ID: r.SetMerchantAction.ID, Name: r.SetMerchantAction.Name}
		}
		for _, t := range r.AddTagsAction {
			rule.AddTagsAction = append(rule.AddTagsAction, RuleTagAction{ID: t.ID, Name: t.Name})
		}
		rules[i] = rule
	}
	return rules, nil
}

func (s *Service) CreateRule(ctx context.Context, input CreateRuleInput) error {
	ruleInput := map[string]interface{}{
		"applyToExistingTransactions": input.ApplyToExisting,
	}
	if input.MerchantOperator != "" && input.MerchantValue != "" {
		ruleInput["merchantNameCriteria"] = []map[string]interface{}{{"operator": input.MerchantOperator, "value": input.MerchantValue}}
	}
	if input.AmountOperator != "" && input.AmountValue != nil {
		ruleInput["amountCriteria"] = map[string]interface{}{"operator": input.AmountOperator, "isExpense": input.AmountIsExpense, "value": *input.AmountValue, "valueRange": nil}
	}
	if input.SetCategoryID != "" {
		ruleInput["setCategoryAction"] = input.SetCategoryID
	}
	if len(input.AddTagIDs) > 0 {
		ruleInput["addTagsAction"] = input.AddTagIDs
	}
	if len(input.AccountIDs) > 0 {
		ruleInput["accountIds"] = input.AccountIDs
	}

	var resp struct {
		CreateTransactionRuleV2 struct {
			Errors *struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"createTransactionRuleV2"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_CreateTransactionRuleMutationV2",
		Query:         CreateTransactionRuleMutation,
		Variables:     map[string]interface{}{"input": ruleInput},
	}, &resp)
	if err != nil {
		return err
	}
	if resp.CreateTransactionRuleV2.Errors != nil && resp.CreateTransactionRuleV2.Errors.Message != "" {
		return errors.New(errors.APIError, resp.CreateTransactionRuleV2.Errors.Message, errors.CatAPI, false, nil)
	}
	return nil
}

func (s *Service) UpdateRule(ctx context.Context, input UpdateRuleInput) error {
	ruleInput := map[string]interface{}{
		"id":                          input.ID,
		"applyToExistingTransactions": input.ApplyToExisting,
	}
	if input.MerchantOperator != "" && input.MerchantValue != "" {
		ruleInput["merchantNameCriteria"] = []map[string]interface{}{{"operator": input.MerchantOperator, "value": input.MerchantValue}}
	}
	if input.AmountOperator != "" && input.AmountValue != nil {
		ruleInput["amountCriteria"] = map[string]interface{}{"operator": input.AmountOperator, "isExpense": input.AmountIsExpense, "value": *input.AmountValue, "valueRange": nil}
	}
	if input.SetCategoryID != "" {
		ruleInput["setCategoryAction"] = input.SetCategoryID
	}
	if len(input.AddTagIDs) > 0 {
		ruleInput["addTagsAction"] = input.AddTagIDs
	}
	if len(input.AccountIDs) > 0 {
		ruleInput["accountIds"] = input.AccountIDs
	}

	var resp struct {
		UpdateTransactionRuleV2 struct {
			Errors *struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"updateTransactionRuleV2"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_UpdateTransactionRuleMutationV2",
		Query:         UpdateTransactionRuleMutation,
		Variables:     map[string]interface{}{"input": ruleInput},
	}, &resp)
	if err != nil {
		return err
	}
	if resp.UpdateTransactionRuleV2.Errors != nil && resp.UpdateTransactionRuleV2.Errors.Message != "" {
		return errors.New(errors.APIError, resp.UpdateTransactionRuleV2.Errors.Message, errors.CatAPI, false, nil)
	}
	return nil
}

func (s *Service) DeleteRule(ctx context.Context, id string) error {
	var resp struct {
		DeleteTransactionRule struct {
			Deleted bool `json:"deleted"`
			Errors  *struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"deleteTransactionRule"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_DeleteTransactionRule",
		Query:         DeleteTransactionRuleMutation,
		Variables:     map[string]interface{}{"id": id},
	}, &resp)
	if err != nil {
		return err
	}
	if !resp.DeleteTransactionRule.Deleted {
		msg := "failed to delete rule"
		if resp.DeleteTransactionRule.Errors != nil {
			msg = resp.DeleteTransactionRule.Errors.Message
		}
		return errors.New(errors.APIError, msg, errors.CatAPI, false, nil)
	}
	return nil
}
