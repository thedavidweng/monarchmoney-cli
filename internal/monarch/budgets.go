package monarch

import (
	"context"
	_ "embed"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

//go:embed queries/budgets/list.graphql
var GetBudgetsQuery string

//go:embed queries/budgets/set.graphql
var SetBudgetMutation string

//go:embed queries/budgets/reset.graphql
var ResetBudgetMutation string

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
