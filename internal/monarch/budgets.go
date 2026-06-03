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

func (s *Service) GetBudget(ctx context.Context, categoryID string, startDate, endDate string) (*Budget, error) {
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

	variables := map[string]interface{}{
		"startDate": startDate,
		"endDate":   endDate,
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetJointPlanningData",
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

	return s.Client.Do(ctx, &graphql.Request{
		OperationName: "UpdateFlexibleBudget",
		Query:         UpdateFlexibleBudgetMutation,
		Variables: map[string]interface{}{
			"input": map[string]interface{}{
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

	return s.Client.Do(ctx, &graphql.Request{
		OperationName: "UpdateFlexRolloverSettings",
		Query:         UpdateFlexRolloverSettingsMutation,
		Variables: map[string]interface{}{
			"input": map[string]interface{}{
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

	variables := map[string]interface{}{
		"startDate": opts.StartDate,
		"endDate":   opts.EndDate,
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetJointPlanningData",
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
		SetBudget struct {
			Budget struct {
				Category struct {
					Name string `json:"name"`
				} `json:"category"`
				Planned float64 `json:"planned"`
			} `json:"budget"`
		} `json:"setBudget"`
	}

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"categoryId": categoryID,
			"amount":     amount,
			"month":      startDate,
		},
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "SetBudget",
		Query:         SetBudgetMutation,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &Budget{
		CategoryName: resp.SetBudget.Budget.Category.Name,
		Planned:      resp.SetBudget.Budget.Planned,
	}, nil
}

func (s *Service) ResetBudget(ctx context.Context, month, year int) error {
	var resp struct {
		ResetBudget struct {
			OK bool `json:"ok"`
		} `json:"resetBudget"`
	}

	return s.Client.Do(ctx, &graphql.Request{
		OperationName: "ResetBudget",
		Query:         ResetBudgetMutation,
		Variables:     map[string]interface{}{"month": month, "year": year},
	}, &resp)
}
