package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetBudgetsQuery = queries.Get("budgets/list.graphql")
var SetBudgetMutation = queries.Get("budgets/set.graphql")
var ResetBudgetMutation = queries.Get("budgets/reset.graphql")
var UpdateFlexibleBudgetMutation = queries.Get("budgets/flexible_set.graphql")
var UpdateFlexRolloverSettingsMutation = queries.Get("budgets/flex_rollover_set.graphql")
var GetBudgetSettingsQuery = queries.Get("budgets/settings.graphql")
var GetFlexRolloverSettingsQuery = queries.Get("budgets/flex_rollover_show.graphql")
var CreateBudgetMutation = queries.Get("budgets/create.graphql")
var ClearBudgetMutation = queries.Get("budgets/clear.graphql")
var ResetBudgetRolloverMutation = queries.Get("budgets/reset_rollover.graphql")

type Budget struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Planned      float64 `json:"planned"`
	Actual       float64 `json:"actual"`
}

type ListBudgetsOptions struct {
	StartDate string
	EndDate   string
}

func (s *Service) GetBudget(ctx context.Context, categoryID, startDate, endDate string) (*Budget, error) {
	var resp struct {
		BudgetData struct {
			MonthlyAmountsByCategory []struct {
				Category struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"category"`
				MonthlyAmounts []struct {
					Month                 string  `json:"month"`
					PlannedCashFlowAmount float64 `json:"plannedCashFlowAmount"`
					ActualAmount          float64 `json:"actualAmount"`
				} `json:"monthlyAmounts"`
			} `json:"monthlyAmountsByCategory"`
		} `json:"budgetData"`
	}

	variables := map[string]any{
		"startDate": startDate,
		"endDate":   endDate,
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_GetJointPlanningData",
		Query:         GetBudgetsQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	for _, cat := range resp.BudgetData.MonthlyAmountsByCategory {
		if cat.Category.ID == categoryID && len(cat.MonthlyAmounts) > 0 {
			return &Budget{
				CategoryID:   cat.Category.ID,
				CategoryName: cat.Category.Name,
				Planned:      cat.MonthlyAmounts[0].PlannedCashFlowAmount,
				Actual:       cat.MonthlyAmounts[0].ActualAmount,
			}, nil
		}
	}

	return nil, nil
}

func (s *Service) UpdateFlexibleBudget(ctx context.Context, month, year int, amount float64) error {
	var resp struct {
		UpdateOrCreateFlexBudgetItem struct {
			FlexBudgetItem struct {
				Month int `json:"month"`
			} `json:"flexBudgetItem"`
		} `json:"updateOrCreateFlexBudgetItem"`
	}

	return s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateFlexBudgetMutation",
		Query:         UpdateFlexibleBudgetMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"month":                 month,
				"year":                  year,
				"plannedCashFlowAmount": amount,
			},
		},
	}, &resp)
}

func (s *Service) UpdateFlexRolloverSettings(ctx context.Context, startMonth string, startingBalance float64, enabled bool) error {
	var resp struct {
		UpdateBudgetSettings struct {
			BudgetRolloverPeriod struct {
				ID string `json:"id"`
			} `json:"budgetRolloverPeriod"`
		} `json:"updateBudgetSettings"`
	}

	return s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "UpdateFlexRolloverSettings",
		Query:         UpdateFlexRolloverSettingsMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"rolloverStartMonth":      startMonth,
				"rolloverStartingBalance": startingBalance,
				"rolloverEnabled":         enabled,
			},
		},
	}, &resp)
}

func (s *Service) ListBudgets(ctx context.Context, opts ListBudgetsOptions) ([]Budget, error) {
	var resp struct {
		BudgetData struct {
			MonthlyAmountsByCategory []struct {
				Category struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"category"`
				MonthlyAmounts []struct {
					Month                 string  `json:"month"`
					PlannedCashFlowAmount float64 `json:"plannedCashFlowAmount"`
					ActualAmount          float64 `json:"actualAmount"`
				} `json:"monthlyAmounts"`
			} `json:"monthlyAmountsByCategory"`
		} `json:"budgetData"`
	}

	variables := map[string]any{
		"startDate": opts.StartDate,
		"endDate":   opts.EndDate,
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_GetJointPlanningData",
		Query:         GetBudgetsQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	budgets := make([]Budget, 0, len(resp.BudgetData.MonthlyAmountsByCategory))
	for _, cat := range resp.BudgetData.MonthlyAmountsByCategory {
		for _, m := range cat.MonthlyAmounts {
			budgets = append(budgets, Budget{
				CategoryID:   cat.Category.ID,
				CategoryName: cat.Category.Name,
				Planned:      m.PlannedCashFlowAmount,
				Actual:       m.ActualAmount,
			})
		}
	}

	return budgets, nil
}

func (s *Service) SetBudget(ctx context.Context, categoryID string, amount float64, startDate string) (*Budget, error) {
	var resp struct {
		UpdateOrCreateBudgetItem struct {
			BudgetItem struct {
				ID           string  `json:"id"`
				BudgetAmount float64 `json:"budgetAmount"`
			} `json:"budgetItem"`
		} `json:"updateOrCreateBudgetItem"`
	}

	variables := map[string]any{
		"input": map[string]any{
			"categoryId":    categoryID,
			"amount":        amount,
			"timeframe":     "month",
			"startDate":     startDate,
			"applyToFuture": false,
		},
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateBudgetItem",
		Query:         SetBudgetMutation,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &Budget{
		CategoryID: categoryID,
		Planned:    resp.UpdateOrCreateBudgetItem.BudgetItem.BudgetAmount,
	}, nil
}

func (s *Service) ResetBudget(ctx context.Context, startDate string, overwriteExisting bool, categoryIDs []string) error {
	input := map[string]any{
		"startDate":         startDate,
		"overwriteExisting": overwriteExisting,
	}
	if len(categoryIDs) > 0 {
		input["filters"] = map[string]any{"categoryIds": categoryIDs}
	}
	var resp struct {
		ResetBudget struct {
			Errors []payloadError `json:"errors"`
		} `json:"resetBudget"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_ResetBudget",
		Query:         ResetBudgetMutation,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.ResetBudget.Errors, "failed to reset budget"); apiErr != nil {
		return apiErr
	}
	return nil
}

type BudgetSettings struct {
	System                     string `json:"system"`
	ApplyToFutureMonthsDefault *bool  `json:"apply_to_future_months_default,omitempty"`
	HasBudget                  bool   `json:"has_budget"`
	HasTransactions            bool   `json:"has_transactions"`
}

type FlexRolloverPeriod struct {
	ID              string  `json:"id"`
	StartMonth      string  `json:"start_month,omitempty"`
	EndMonth        string  `json:"end_month,omitempty"`
	StartingBalance float64 `json:"starting_balance,omitempty"`
	Frequency       string  `json:"frequency,omitempty"`
	TargetAmount    float64 `json:"target_amount,omitempty"`
	Type            string  `json:"type,omitempty"`
}

func (s *Service) GetBudgetSettings(ctx context.Context) (*BudgetSettings, error) {
	var resp struct {
		BudgetSystem                     string `json:"budgetSystem"`
		BudgetApplyToFutureMonthsDefault *bool  `json:"budgetApplyToFutureMonthsDefault"`
		BudgetStatus                     *struct {
			HasBudget       bool `json:"hasBudget"`
			HasTransactions bool `json:"hasTransactions"`
		} `json:"budgetStatus"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_BudgetSettings",
		Query:         GetBudgetSettingsQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}
	settings := &BudgetSettings{
		System:                     resp.BudgetSystem,
		ApplyToFutureMonthsDefault: resp.BudgetApplyToFutureMonthsDefault,
	}
	if resp.BudgetStatus != nil {
		settings.HasBudget = resp.BudgetStatus.HasBudget
		settings.HasTransactions = resp.BudgetStatus.HasTransactions
	}
	return settings, nil
}

func (s *Service) GetFlexRolloverSettings(ctx context.Context) (*FlexRolloverPeriod, error) {
	var resp struct {
		FlexExpenseRolloverPeriod *struct {
			ID              string  `json:"id"`
			StartMonth      string  `json:"startMonth"`
			EndMonth        string  `json:"endMonth"`
			StartingBalance float64 `json:"startingBalance"`
			Frequency       string  `json:"frequency"`
			TargetAmount    float64 `json:"targetAmount"`
			Type            string  `json:"type"`
		} `json:"flexExpenseRolloverPeriod"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_GetFlexibleGroupRolloverSettings",
		Query:         GetFlexRolloverSettingsQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.FlexExpenseRolloverPeriod == nil {
		return &FlexRolloverPeriod{}, nil
	}
	p := resp.FlexExpenseRolloverPeriod
	return &FlexRolloverPeriod{
		ID: p.ID, StartMonth: p.StartMonth, EndMonth: p.EndMonth,
		StartingBalance: p.StartingBalance, Frequency: p.Frequency,
		TargetAmount: p.TargetAmount, Type: p.Type,
	}, nil
}

func (s *Service) SetBudgetGroup(ctx context.Context, groupID string, amount float64, startDate string) error {
	var resp struct {
		UpdateOrCreateBudgetItem struct {
			Errors []payloadError `json:"errors"`
		} `json:"updateOrCreateBudgetItem"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateBudgetItem",
		Query:         SetBudgetMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"categoryGroupId": groupID,
				"amount":          amount,
				"timeframe":       "month",
				"startDate":       startDate,
				"applyToFuture":   false,
			},
		},
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.UpdateOrCreateBudgetItem.Errors, "failed to set group budget"); apiErr != nil {
		return apiErr
	}
	return nil
}

func (s *Service) CreateBudget(ctx context.Context, startDate string) error {
	var resp struct {
		CreateBudget struct {
			Errors []payloadError `json:"errors"`
		} `json:"createBudget"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_CreateBudgetForHousehold",
		Query:         CreateBudgetMutation,
		Variables: map[string]any{
			"input": map[string]any{"startDate": startDate, "timeframe": "month"},
		},
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.CreateBudget.Errors, "failed to create budget"); apiErr != nil {
		return apiErr
	}
	return nil
}

func (s *Service) ClearBudget(ctx context.Context, startDate string) error {
	var resp struct {
		ClearBudget struct {
			Errors []payloadError `json:"errors"`
		} `json:"clearBudget"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_ClearAllMutation",
		Query:         ClearBudgetMutation,
		Variables:     map[string]any{"input": map[string]any{"startDate": startDate}},
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.ClearBudget.Errors, "failed to clear budget"); apiErr != nil {
		return apiErr
	}
	return nil
}

type ResetRolloverOptions struct {
	StartMonth      string
	CategoryID      string
	CategoryGroupID string
	StartingBalance *float64
}

func (s *Service) ResetBudgetRollover(ctx context.Context, opts *ResetRolloverOptions) error {
	input := map[string]any{"startMonth": opts.StartMonth}
	if opts.CategoryID != "" {
		input["categoryId"] = opts.CategoryID
	}
	if opts.CategoryGroupID != "" {
		input["categoryGroupId"] = opts.CategoryGroupID
	}
	if opts.StartingBalance != nil {
		input["startingBalance"] = *opts.StartingBalance
	}
	var resp struct {
		ResetBudgetRollover struct {
			Errors []payloadError `json:"errors"`
		} `json:"resetBudgetRollover"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_ResetRolloverMutation",
		Query:         ResetBudgetRolloverMutation,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.ResetBudgetRollover.Errors, "failed to reset budget rollover"); apiErr != nil {
		return apiErr
	}
	return nil
}
