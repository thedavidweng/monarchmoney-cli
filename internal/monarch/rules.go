package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var (
	GetTransactionRulesQuery       = queries.Get("rules/list.graphql")
	CreateTransactionRuleMutation  = queries.Get("rules/create.graphql")
	UpdateTransactionRuleMutation  = queries.Get("rules/update.graphql")
	DeleteTransactionRuleMutation  = queries.Get("rules/delete.graphql")
	ReorderTransactionRuleMutation = queries.Get("rules/reorder.graphql")
)

type RuleCriteria struct {
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type RuleAmountRange struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
}

type RuleAmountCriteria struct {
	Operator   string           `json:"operator"`
	IsExpense  bool             `json:"is_expense"`
	Value      float64          `json:"value"`
	ValueRange *RuleAmountRange `json:"value_range,omitempty"`
}

type RuleAction struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RuleTagAction struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RuleSplitAction struct {
	AmountType string           `json:"amountType"`
	SplitsInfo []map[string]any `json:"splitsInfo"`
}

type Rule struct {
	ID                                   string              `json:"id"`
	Order                                int                 `json:"order"`
	MerchantCriteriaUseOriginalStatement bool                `json:"merchant_criteria_use_original_statement"`
	MerchantCriteria                     []RuleCriteria      `json:"merchant_criteria,omitempty"`
	MerchantNameCriteria                 []RuleCriteria      `json:"merchant_name_criteria,omitempty"`
	OriginalStatementCriteria            []RuleCriteria      `json:"original_statement_criteria,omitempty"`
	AmountCriteria                       *RuleAmountCriteria `json:"amount_criteria,omitempty"`
	CategoryIDs                          []string            `json:"category_ids,omitempty"`
	Categories                           []RuleAction        `json:"categories,omitempty"`
	AccountIDs                           []string            `json:"account_ids,omitempty"`
	Accounts                             []RuleAction        `json:"accounts,omitempty"`
	CriteriaOwnerIsJoint                 *bool               `json:"criteria_owner_is_joint,omitempty"`
	CriteriaOwnerUserIDs                 []string            `json:"criteria_owner_user_ids,omitempty"`
	CriteriaOwnerUsers                   []RuleAction        `json:"criteria_owner_users,omitempty"`
	CriteriaBusinessEntityIDs            []string            `json:"criteria_business_entity_ids,omitempty"`
	CriteriaBusinessEntityIsUnassigned   *bool               `json:"criteria_business_entity_is_unassigned,omitempty"`
	CriteriaBusinessEntities             []RuleAction        `json:"criteria_business_entities,omitempty"`
	SetCategoryAction                    *RuleAction         `json:"set_category_action,omitempty"`
	SetMerchantAction                    *RuleAction         `json:"set_merchant_action,omitempty"`
	AddTagsAction                        []RuleTagAction     `json:"add_tags_action,omitempty"`
	LinkGoalAction                       *RuleAction         `json:"link_goal_action,omitempty"`
	LinkSavingsGoalAction                *RuleAction         `json:"link_savings_goal_action,omitempty"`
	NeedsReviewByUserAction              *RuleAction         `json:"needs_review_by_user_action,omitempty"`
	UnassignNeedsReviewByUserAction      *bool               `json:"unassign_needs_review_by_user_action,omitempty"`
	SendNotificationAction               *bool               `json:"send_notification_action,omitempty"`
	SetHideFromReportsAction             *bool               `json:"set_hide_from_reports_action,omitempty"`
	SetLinkToPaydownBudgetAction         *bool               `json:"set_link_to_paydown_budget_action,omitempty"`
	ReviewStatusAction                   *string             `json:"review_status_action,omitempty"`
	ActionSetOwnerIsJoint                *bool               `json:"action_set_owner_is_joint,omitempty"`
	ActionSetOwner                       *RuleAction         `json:"action_set_owner,omitempty"`
	ActionSetBusinessEntity              *RuleAction         `json:"action_set_business_entity,omitempty"`
	ActionSetBusinessEntityIsUnassigned  *bool               `json:"action_set_business_entity_is_unassigned,omitempty"`
	SplitTransactionsAction              *RuleSplitAction    `json:"split_transactions_action,omitempty"`
	RecentApplicationCount               int                 `json:"recent_application_count"`
	LastAppliedAt                        string              `json:"last_applied_at,omitempty"`
}

type RuleFields struct {
	MerchantOperator                    string
	MerchantValue                       string
	LegacyMerchantOperator              string
	LegacyMerchantValue                 string
	UseOriginalStatement                *bool
	OriginalStatementOperator           string
	OriginalStatementValue              string
	AmountOperator                      string
	AmountValue                         *float64
	AmountValueUpper                    *float64
	AmountIsExpense                     bool
	CategoryIDs                         []string
	AccountIDs                          []string
	CriteriaOwnerUserIDs                []string
	CriteriaOwnerIsJoint                *bool
	CriteriaBusinessEntityIDs           []string
	CriteriaBusinessEntityIsUnassigned  *bool
	SetCategoryID                       string
	SetMerchant                         string
	AddTagIDs                           []string
	HideFromReports                     *bool
	ReviewStatus                        string
	NeedsReviewByUserID                 string
	LinkGoalID                          string
	LinkSavingsGoalID                   string
	LinkToPaydownBudget                 *bool
	SendNotification                    *bool
	ActionSetOwner                      string
	ActionSetOwnerIsJoint               *bool
	ActionSetBusinessEntity             string
	ActionSetBusinessEntityIsUnassigned *bool
	SplitAction                         *RuleSplitAction
	ApplyToExisting                     bool
}

type CreateRuleInput struct {
	RuleFields
}

type UpdateRuleInput struct {
	ID string
	RuleFields
}

func criterionInput(operator, value string) []map[string]any {
	return []map[string]any{{"operator": operator, "value": value}}
}

func buildRuleInput(ruleInput map[string]any, f *RuleFields) {
	if f.MerchantOperator != "" && f.MerchantValue != "" {
		ruleInput["merchantNameCriteria"] = criterionInput(f.MerchantOperator, f.MerchantValue)
	}
	if f.LegacyMerchantOperator != "" && f.LegacyMerchantValue != "" {
		ruleInput["merchantCriteria"] = criterionInput(f.LegacyMerchantOperator, f.LegacyMerchantValue)
	}
	if f.UseOriginalStatement != nil {
		ruleInput["merchantCriteriaUseOriginalStatement"] = *f.UseOriginalStatement
	}
	if f.OriginalStatementOperator != "" && f.OriginalStatementValue != "" {
		ruleInput["originalStatementCriteria"] = criterionInput(f.OriginalStatementOperator, f.OriginalStatementValue)
	}
	if f.AmountOperator != "" && f.AmountValue != nil {
		amount := map[string]any{"operator": f.AmountOperator, "isExpense": f.AmountIsExpense, "value": *f.AmountValue, "valueRange": nil}
		if f.AmountOperator == "between" && f.AmountValueUpper != nil {
			amount["valueRange"] = map[string]any{"lower": *f.AmountValue, "upper": *f.AmountValueUpper}
		}
		ruleInput["amountCriteria"] = amount
	}
	if len(f.CategoryIDs) > 0 {
		ruleInput["categoryIds"] = f.CategoryIDs
	}
	if len(f.AccountIDs) > 0 {
		ruleInput["accountIds"] = f.AccountIDs
	}
	if len(f.CriteriaOwnerUserIDs) > 0 {
		ruleInput["criteriaOwnerUserIds"] = f.CriteriaOwnerUserIDs
	}
	if f.CriteriaOwnerIsJoint != nil {
		ruleInput["criteriaOwnerIsJoint"] = *f.CriteriaOwnerIsJoint
	}
	if len(f.CriteriaBusinessEntityIDs) > 0 {
		ruleInput["criteriaBusinessEntityIds"] = f.CriteriaBusinessEntityIDs
	}
	if f.CriteriaBusinessEntityIsUnassigned != nil {
		ruleInput["criteriaBusinessEntityIsUnassigned"] = *f.CriteriaBusinessEntityIsUnassigned
	}
	if f.SetCategoryID != "" {
		ruleInput["setCategoryAction"] = f.SetCategoryID
	}
	if f.SetMerchant != "" {
		ruleInput["setMerchantAction"] = f.SetMerchant
	}
	if len(f.AddTagIDs) > 0 {
		ruleInput["addTagsAction"] = f.AddTagIDs
	}
	if f.HideFromReports != nil {
		ruleInput["setHideFromReportsAction"] = *f.HideFromReports
	}
	if f.ReviewStatus != "" {
		ruleInput["reviewStatusAction"] = f.ReviewStatus
	}
	if f.NeedsReviewByUserID != "" {
		ruleInput["needsReviewByUserAction"] = f.NeedsReviewByUserID
	}
	if f.LinkGoalID != "" {
		ruleInput["linkGoalAction"] = f.LinkGoalID
	}
	if f.LinkSavingsGoalID != "" {
		ruleInput["linkSavingsGoalAction"] = f.LinkSavingsGoalID
	}
	if f.LinkToPaydownBudget != nil {
		ruleInput["setLinkToPaydownBudgetAction"] = *f.LinkToPaydownBudget
	}
	if f.SendNotification != nil {
		ruleInput["sendNotificationAction"] = *f.SendNotification
	}
	if f.ActionSetOwner != "" {
		ruleInput["actionSetOwner"] = f.ActionSetOwner
	}
	if f.ActionSetOwnerIsJoint != nil {
		ruleInput["actionSetOwnerIsJoint"] = *f.ActionSetOwnerIsJoint
	}
	if f.ActionSetBusinessEntity != "" {
		ruleInput["actionSetBusinessEntity"] = f.ActionSetBusinessEntity
	}
	if f.ActionSetBusinessEntityIsUnassigned != nil {
		ruleInput["actionSetBusinessEntityIsUnassigned"] = *f.ActionSetBusinessEntityIsUnassigned
	}
	if f.SplitAction != nil {
		ruleInput["splitTransactionsAction"] = map[string]any{"amountType": f.SplitAction.AmountType, "splitsInfo": f.SplitAction.SplitsInfo}
	}
	ruleInput["applyToExistingTransactions"] = f.ApplyToExisting
}

func toRuleCriteria(in []struct {
	Operator string `json:"operator"`
	Value    string `json:"value"`
},
) []RuleCriteria {
	out := make([]RuleCriteria, len(in))
	for i := range in {
		out[i] = RuleCriteria{Operator: in[i].Operator, Value: in[i].Value}
	}
	return out
}

func (s *Service) ListRules(ctx context.Context) ([]Rule, error) {
	var resp struct {
		TransactionRules []struct {
			ID                                   string `json:"id"`
			Order                                int    `json:"order"`
			MerchantCriteriaUseOriginalStatement bool   `json:"merchantCriteriaUseOriginalStatement"`
			MerchantCriteria                     []struct {
				Operator string `json:"operator"`
				Value    string `json:"value"`
			} `json:"merchantCriteria"`
			MerchantNameCriteria []struct {
				Operator string `json:"operator"`
				Value    string `json:"value"`
			} `json:"merchantNameCriteria"`
			OriginalStatementCriteria []struct {
				Operator string `json:"operator"`
				Value    string `json:"value"`
			} `json:"originalStatementCriteria"`
			AmountCriteria *struct {
				Operator   string  `json:"operator"`
				IsExpense  bool    `json:"isExpense"`
				Value      float64 `json:"value"`
				ValueRange *struct {
					Lower float64 `json:"lower"`
					Upper float64 `json:"upper"`
				} `json:"valueRange"`
			} `json:"amountCriteria"`
			CategoryIDs []string `json:"categoryIds"`
			Categories  []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"categories"`
			AccountIDs []string `json:"accountIds"`
			Accounts   []struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"accounts"`
			CriteriaOwnerIsJoint *bool    `json:"criteriaOwnerIsJoint"`
			CriteriaOwnerUserIDs []string `json:"criteriaOwnerUserIds"`
			CriteriaOwnerUsers   []struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"criteriaOwnerUsers"`
			CriteriaBusinessEntityIDs          []string `json:"criteriaBusinessEntityIds"`
			CriteriaBusinessEntityIsUnassigned *bool    `json:"criteriaBusinessEntityIsUnassigned"`
			CriteriaBusinessEntities           []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"criteriaBusinessEntities"`
			SetCategoryAction *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"setCategoryAction"`
			SetMerchantAction *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"setMerchantAction"`
			AddTagsAction []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Color string `json:"color"`
			} `json:"addTagsAction"`
			LinkGoalAction *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"linkGoalAction"`
			LinkSavingsGoalAction *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"linkSavingsGoalAction"`
			NeedsReviewByUserAction *struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"needsReviewByUserAction"`
			UnassignNeedsReviewByUserAction *bool   `json:"unassignNeedsReviewByUserAction"`
			SendNotificationAction          *bool   `json:"sendNotificationAction"`
			SetHideFromReportsAction        *bool   `json:"setHideFromReportsAction"`
			SetLinkToPaydownBudgetAction    *bool   `json:"setLinkToPaydownBudgetAction"`
			ReviewStatusAction              *string `json:"reviewStatusAction"`
			ActionSetOwnerIsJoint           *bool   `json:"actionSetOwnerIsJoint"`
			ActionSetOwner                  *struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"actionSetOwner"`
			ActionSetBusinessEntity *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"actionSetBusinessEntity"`
			ActionSetBusinessEntityIsUnassigned *bool `json:"actionSetBusinessEntityIsUnassigned"`
			SplitTransactionsAction             *struct {
				AmountType string           `json:"amountType"`
				SplitsInfo []map[string]any `json:"splitsInfo"`
			} `json:"splitTransactionsAction"`
			RecentApplicationCount int    `json:"recentApplicationCount"`
			LastAppliedAt          string `json:"lastAppliedAt"`
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
	for i := range resp.TransactionRules {
		r := &resp.TransactionRules[i]
		rule := Rule{
			ID:                                   r.ID,
			Order:                                r.Order,
			MerchantCriteriaUseOriginalStatement: r.MerchantCriteriaUseOriginalStatement,
			CategoryIDs:                          r.CategoryIDs,
			AccountIDs:                           r.AccountIDs,
			CriteriaOwnerIsJoint:                 r.CriteriaOwnerIsJoint,
			CriteriaOwnerUserIDs:                 r.CriteriaOwnerUserIDs,
			CriteriaBusinessEntityIDs:            r.CriteriaBusinessEntityIDs,
			CriteriaBusinessEntityIsUnassigned:   r.CriteriaBusinessEntityIsUnassigned,
			ReviewStatusAction:                   r.ReviewStatusAction,
			UnassignNeedsReviewByUserAction:      r.UnassignNeedsReviewByUserAction,
			SendNotificationAction:               r.SendNotificationAction,
			SetHideFromReportsAction:             r.SetHideFromReportsAction,
			SetLinkToPaydownBudgetAction:         r.SetLinkToPaydownBudgetAction,
			ActionSetOwnerIsJoint:                r.ActionSetOwnerIsJoint,
			ActionSetBusinessEntityIsUnassigned:  r.ActionSetBusinessEntityIsUnassigned,
			RecentApplicationCount:               r.RecentApplicationCount,
			LastAppliedAt:                        r.LastAppliedAt,
		}
		rule.MerchantCriteria = toRuleCriteria(r.MerchantCriteria)
		rule.MerchantNameCriteria = toRuleCriteria(r.MerchantNameCriteria)
		rule.OriginalStatementCriteria = toRuleCriteria(r.OriginalStatementCriteria)
		if len(rule.MerchantCriteria) == 0 {
			rule.MerchantCriteria = nil
		}
		if len(rule.MerchantNameCriteria) == 0 {
			rule.MerchantNameCriteria = nil
		}
		if len(rule.OriginalStatementCriteria) == 0 {
			rule.OriginalStatementCriteria = nil
		}
		if r.AmountCriteria != nil {
			rule.AmountCriteria = &RuleAmountCriteria{Operator: r.AmountCriteria.Operator, IsExpense: r.AmountCriteria.IsExpense, Value: r.AmountCriteria.Value}
			if r.AmountCriteria.ValueRange != nil {
				rule.AmountCriteria.ValueRange = &RuleAmountRange{Lower: r.AmountCriteria.ValueRange.Lower, Upper: r.AmountCriteria.ValueRange.Upper}
			}
		}
		rule.Categories = nil
		for _, c := range r.Categories {
			rule.Categories = append(rule.Categories, RuleAction{ID: c.ID, Name: c.Name})
		}
		rule.Accounts = nil
		for _, a := range r.Accounts {
			rule.Accounts = append(rule.Accounts, RuleAction{ID: a.ID, Name: a.DisplayName})
		}
		rule.CriteriaOwnerUsers = nil
		for _, u := range r.CriteriaOwnerUsers {
			rule.CriteriaOwnerUsers = append(rule.CriteriaOwnerUsers, RuleAction{ID: u.ID, Name: u.DisplayName})
		}
		rule.CriteriaBusinessEntities = nil
		for _, b := range r.CriteriaBusinessEntities {
			rule.CriteriaBusinessEntities = append(rule.CriteriaBusinessEntities, RuleAction{ID: b.ID, Name: b.Name})
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
		if r.LinkGoalAction != nil {
			rule.LinkGoalAction = &RuleAction{ID: r.LinkGoalAction.ID, Name: r.LinkGoalAction.Name}
		}
		if r.LinkSavingsGoalAction != nil {
			rule.LinkSavingsGoalAction = &RuleAction{ID: r.LinkSavingsGoalAction.ID, Name: r.LinkSavingsGoalAction.Name}
		}
		if r.NeedsReviewByUserAction != nil {
			rule.NeedsReviewByUserAction = &RuleAction{ID: r.NeedsReviewByUserAction.ID, Name: r.NeedsReviewByUserAction.DisplayName}
		}
		if r.ActionSetOwner != nil {
			rule.ActionSetOwner = &RuleAction{ID: r.ActionSetOwner.ID, Name: r.ActionSetOwner.DisplayName}
		}
		if r.ActionSetBusinessEntity != nil {
			rule.ActionSetBusinessEntity = &RuleAction{ID: r.ActionSetBusinessEntity.ID, Name: r.ActionSetBusinessEntity.Name}
		}
		if r.SplitTransactionsAction != nil {
			rule.SplitTransactionsAction = &RuleSplitAction{AmountType: r.SplitTransactionsAction.AmountType, SplitsInfo: r.SplitTransactionsAction.SplitsInfo}
		}
		rules[i] = rule
	}
	return rules, nil
}

func (s *Service) CreateRule(ctx context.Context, input *CreateRuleInput) error {
	ruleInput := map[string]any{}
	buildRuleInput(ruleInput, &input.RuleFields)

	var resp struct {
		CreateTransactionRuleV2 struct {
			Errors *struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"createTransactionRuleV2"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_CreateTransactionRuleMutationV2",
		Query:         CreateTransactionRuleMutation,
		Variables:     map[string]any{"input": ruleInput},
	}, &resp)
	if err != nil {
		return err
	}
	if resp.CreateTransactionRuleV2.Errors != nil && resp.CreateTransactionRuleV2.Errors.Message != "" {
		return errors.New(errors.APIError, resp.CreateTransactionRuleV2.Errors.Message, errors.CatAPI, false, nil)
	}
	return nil
}

func (s *Service) UpdateRule(ctx context.Context, input *UpdateRuleInput) error {
	ruleInput := map[string]any{
		"id": input.ID,
	}
	buildRuleInput(ruleInput, &input.RuleFields)

	var resp struct {
		UpdateTransactionRuleV2 struct {
			Errors *struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"updateTransactionRuleV2"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateTransactionRuleMutationV2",
		Query:         UpdateTransactionRuleMutation,
		Variables:     map[string]any{"input": ruleInput},
	}, &resp)
	if err != nil {
		return err
	}
	if resp.UpdateTransactionRuleV2.Errors != nil && resp.UpdateTransactionRuleV2.Errors.Message != "" {
		return errors.New(errors.APIError, resp.UpdateTransactionRuleV2.Errors.Message, errors.CatAPI, false, nil)
	}
	return nil
}

type RuleOrderEntry struct {
	ID    string `json:"id"`
	Order int    `json:"order"`
}

type RuleReorderResult struct {
	RuleID    string           `json:"rule_id"`
	MovedFrom int              `json:"moved_from"`
	MovedTo   int              `json:"moved_to"`
	Requested int              `json:"requested_order"`
	Order     []RuleOrderEntry `json:"order"`
}

func (s *Service) ReorderRule(ctx context.Context, id string, order int) (*RuleReorderResult, error) {
	rules, err := s.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	from := -1
	for i := range rules {
		if rules[i].ID == id {
			from = rules[i].Order
			break
		}
	}
	if from < 0 {
		return nil, errors.New(errors.ResourceNotFound, "transaction rule not found", errors.CatAPI, false, nil)
	}

	var resp struct {
		UpdateTransactionRuleOrderV2 struct {
			TransactionRules []struct {
				ID    string `json:"id"`
				Order int    `json:"order"`
			} `json:"transactionRules"`
		} `json:"updateTransactionRuleOrderV2"`
	}

	err = s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_UpdateRuleOrderMutation",
		Query:         ReorderTransactionRuleMutation,
		Variables:     map[string]any{"id": id, "order": order},
	}, &resp)
	if err != nil {
		return nil, err
	}
	result := &RuleReorderResult{RuleID: id, MovedFrom: from, MovedTo: order, Requested: order}
	for _, r := range resp.UpdateTransactionRuleOrderV2.TransactionRules {
		result.Order = append(result.Order, RuleOrderEntry{ID: r.ID, Order: r.Order})
		if r.ID == id {
			result.MovedTo = r.Order
		}
	}
	if result.Order == nil {
		result.Order = []RuleOrderEntry{}
	}
	return result, nil
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

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_DeleteTransactionRule",
		Query:         DeleteTransactionRuleMutation,
		Variables:     map[string]any{"id": id},
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
