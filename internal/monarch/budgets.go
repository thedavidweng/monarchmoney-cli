package monarch

import (
	"context"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
	"github.com/monarchmoney-cli/monarch/queries"
)

var GetBudgetsQuery = queries.Get("budgets/list.graphql")
var GetBudgetQuery = queries.Get("budgets/show.graphql")
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
	Month int
	Year  int
}

func (s *Service) GetBudget(ctx context.Context, categoryID string, month, year int) (*Budget, error) {
	var resp struct {
		Budget struct {
			Category struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"category"`
			Planned float64 `json:"planned"`
			Actual  float64 `json:"actual"`
		} `json:"budget"`
	}

	variables := map[string]interface{}{
		"categoryId": categoryID,
	}
	if month > 0 {
		variables["month"] = month
	}
	if year > 0 {
		variables["year"] = year
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetBudget",
		Query:         GetBudgetQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &Budget{
		CategoryID:   resp.Budget.Category.ID,
		CategoryName: resp.Budget.Category.Name,
		Planned:      resp.Budget.Planned,
		Actual:       resp.Budget.Actual,
	}, nil
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
				"month":                  month,
				"year":                   year,
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
				"rolloverStartMonth":     startMonth,
				"rolloverStartingBalance": startingBalance,
				"rolloverEnabled":         enabled,
			},
		},
	}, &resp)
}

func (s *Service) ListBudgets(ctx context.Context, opts ListBudgetsOptions) ([]Budget, error) {
	var resp struct {
		Budgets []struct {
			Category struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"category"`
			Planned float64 `json:"planned"`
			Actual  float64 `json:"actual"`
		} `json:"budgets"`
	}

	variables := make(map[string]interface{})
	if opts.Month > 0 {
		variables["month"] = opts.Month
	}
	if opts.Year > 0 {
		variables["year"] = opts.Year
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetBudgets",
		Query:         GetBudgetsQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	budgets := make([]Budget, len(resp.Budgets))
	for i, b := range resp.Budgets {
		budgets[i] = Budget{
			CategoryID:   b.Category.ID,
			CategoryName: b.Category.Name,
			Planned:      b.Planned,
			Actual:       b.Actual,
		}
	}

	return budgets, nil
}

func (s *Service) SetBudget(ctx context.Context, categoryID string, amount float64, month, year int) (*Budget, error) {
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
		"categoryId": categoryID,
		"amount":     amount,
	}
	if month > 0 {
		variables["month"] = month
	}
	if year > 0 {
		variables["year"] = year
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
