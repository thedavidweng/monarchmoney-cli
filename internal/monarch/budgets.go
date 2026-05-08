package monarch

import (
	"context"
	_ "embed"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

//go:embed queries/budgets/list.graphql
var GetBudgetsQuery string

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
