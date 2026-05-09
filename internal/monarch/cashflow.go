package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetCashflowSummaryQuery = queries.Get("cashflow/summary.graphql")
var GetCashflowQuery = queries.Get("cashflow/list.graphql")
var GetCashflowCategoriesQuery = queries.Get("cashflow/categories.graphql")
var GetCashflowMerchantsQuery = queries.Get("cashflow/merchants.graphql")

type CashflowSummary struct {
	Income      float64 `json:"income"`
	Expense     float64 `json:"expense"`
	Savings     float64 `json:"savings"`
	SavingsRate float64 `json:"savings_rate"`
}

type CashflowRecord struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

type CashflowPeriod struct {
	Period  string  `json:"period"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Savings float64 `json:"savings"`
}

func (s *Service) ListCashflow(ctx context.Context, startDate, endDate string) ([]CashflowPeriod, error) {
	var resp struct {
		Cashflow struct {
			ByPeriod []struct {
				Period  string  `json:"period"`
				Income  float64 `json:"income"`
				Expense float64 `json:"expense"`
				Savings float64 `json:"savings"`
			} `json:"byPeriod"`
		} `json:"cashflow"`
	}

	variables := map[string]interface{}{
		"startDate": startDate,
		"endDate":   endDate,
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCashflow",
		Query:         GetCashflowQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	periods := make([]CashflowPeriod, len(resp.Cashflow.ByPeriod))
	for i, period := range resp.Cashflow.ByPeriod {
		periods[i] = CashflowPeriod{
			Period:  period.Period,
			Income:  period.Income,
			Expense: period.Expense,
			Savings: period.Savings,
		}
	}

	return periods, nil
}

func (s *Service) GetCashflowSummary(ctx context.Context, startDate, endDate string) (*CashflowSummary, error) {
	var resp struct {
		Aggregates []struct {
			Summary struct {
				SumIncome   float64 `json:"sumIncome"`
				SumExpense  float64 `json:"sumExpense"`
				Savings     float64 `json:"savings"`
				SavingsRate float64 `json:"savingsRate"`
			} `json:"summary"`
		} `json:"aggregates"`
	}

	variables := map[string]interface{}{
		"filters": map[string]interface{}{
			"startDate":  startDate,
			"endDate":    endDate,
			"search":     "",
			"categories": []string{},
			"accounts":   []string{},
			"tags":       []string{},
		},
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCashflowSummary",
		Query:         GetCashflowSummaryQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	if len(resp.Aggregates) == 0 {
		return &CashflowSummary{}, nil
	}

	return &CashflowSummary{
		Income:      resp.Aggregates[0].Summary.SumIncome,
		Expense:     resp.Aggregates[0].Summary.SumExpense,
		Savings:     resp.Aggregates[0].Summary.Savings,
		SavingsRate: resp.Aggregates[0].Summary.SavingsRate,
	}, nil
}

func (s *Service) GetCashflowCategories(ctx context.Context, startDate, endDate string) ([]CashflowRecord, error) {
	var resp struct {
		Aggregates []struct {
			GroupBy struct {
				Category struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"category"`
			} `json:"groupBy"`
			Summary struct {
				Sum float64 `json:"sum"`
			} `json:"summary"`
		} `json:"aggregates"`
	}

	variables := map[string]interface{}{
		"filters": map[string]interface{}{
			"startDate":  startDate,
			"endDate":    endDate,
			"search":     "",
			"categories": []string{},
			"accounts":   []string{},
			"tags":       []string{},
		},
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCashflowCategories",
		Query:         GetCashflowCategoriesQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	records := make([]CashflowRecord, 0, len(resp.Aggregates))
	for _, a := range resp.Aggregates {
		if a.GroupBy.Category.Name != "" {
			records = append(records, CashflowRecord{
				Name:   a.GroupBy.Category.Name,
				Amount: a.Summary.Sum,
			})
		}
	}

	return records, nil
}

func (s *Service) GetCashflowMerchants(ctx context.Context, startDate, endDate string) ([]CashflowRecord, error) {
	var resp struct {
		Aggregates []struct {
			GroupBy struct {
				Merchant struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"merchant"`
			} `json:"groupBy"`
			Summary struct {
				SumIncome  float64 `json:"sumIncome"`
				SumExpense float64 `json:"sumExpense"`
			} `json:"summary"`
		} `json:"aggregates"`
	}

	variables := map[string]interface{}{
		"filters": map[string]interface{}{
			"startDate":  startDate,
			"endDate":    endDate,
			"search":     "",
			"categories": []string{},
			"accounts":   []string{},
			"tags":       []string{},
		},
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCashflowMerchants",
		Query:         GetCashflowMerchantsQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	records := make([]CashflowRecord, 0, len(resp.Aggregates))
	for _, a := range resp.Aggregates {
		if a.GroupBy.Merchant.Name != "" {
			records = append(records, CashflowRecord{
				Name:   a.GroupBy.Merchant.Name,
				Amount: a.Summary.SumExpense,
			})
		}
	}

	return records, nil
}
