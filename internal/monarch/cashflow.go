package monarch

import (
	"context"
	_ "embed"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

//go:embed queries/cashflow/summary.graphql
var GetCashflowSummaryQuery string

//go:embed queries/cashflow/list.graphql
var GetCashflowQuery string

//go:embed queries/cashflow/categories.graphql
var GetCashflowCategoriesQuery string

//go:embed queries/cashflow/merchants.graphql
var GetCashflowMerchantsQuery string

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

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCashflow",
		Query:         GetCashflowQuery,
		Variables: map[string]interface{}{
			"startDate": startDate,
			"endDate":   endDate,
		},
	}, &resp)

	if err != nil {
		return nil, err
	}

	periods := make([]CashflowPeriod, len(resp.Cashflow.ByPeriod))
	for i, p := range resp.Cashflow.ByPeriod {
		periods[i] = CashflowPeriod{
			Period:  p.Period,
			Income:  p.Income,
			Expense: p.Expense,
			Savings: p.Savings,
		}
	}

	return periods, nil
}

func (s *Service) GetCashflowSummary(ctx context.Context, startDate, endDate string) (*CashflowSummary, error) {
	var resp struct {
		CashflowSummary struct {
			Income      float64 `json:"income"`
			Expense     float64 `json:"expense"`
			Savings     float64 `json:"savings"`
			SavingsRate float64 `json:"savingsRate"`
		} `json:"cashflowSummary"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCashflowSummary",
		Query:         GetCashflowSummaryQuery,
		Variables: map[string]interface{}{
			"startDate": startDate,
			"endDate":   endDate,
		},
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &CashflowSummary{
		Income:      resp.CashflowSummary.Income,
		Expense:     resp.CashflowSummary.Expense,
		Savings:     resp.CashflowSummary.Savings,
		SavingsRate: resp.CashflowSummary.SavingsRate,
	}, nil
}

func (s *Service) GetCashflowCategories(ctx context.Context, startDate, endDate string) ([]CashflowRecord, error) {
	var resp struct {
		CashflowCategories []struct {
			Category struct {
				Name string `json:"name"`
			} `json:"category"`
			Amount float64 `json:"amount"`
		} `json:"cashflowCategories"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCashflowCategories",
		Query:         GetCashflowCategoriesQuery,
		Variables: map[string]interface{}{
			"startDate": startDate,
			"endDate":   endDate,
		},
	}, &resp)

	if err != nil {
		return nil, err
	}

	records := make([]CashflowRecord, len(resp.CashflowCategories))
	for i, r := range resp.CashflowCategories {
		records[i] = CashflowRecord{
			Name:   r.Category.Name,
			Amount: r.Amount,
		}
	}

	return records, nil
}

func (s *Service) GetCashflowMerchants(ctx context.Context, startDate, endDate string) ([]CashflowRecord, error) {
	var resp struct {
		CashflowMerchants []struct {
			Merchant struct {
				Name string `json:"name"`
			} `json:"merchant"`
			Amount float64 `json:"amount"`
		} `json:"cashflowMerchants"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCashflowMerchants",
		Query:         GetCashflowMerchantsQuery,
		Variables: map[string]interface{}{
			"startDate": startDate,
			"endDate":   endDate,
		},
	}, &resp)

	if err != nil {
		return nil, err
	}

	records := make([]CashflowRecord, len(resp.CashflowMerchants))
	for i, r := range resp.CashflowMerchants {
		records[i] = CashflowRecord{
			Name:   r.Merchant.Name,
			Amount: r.Amount,
		}
	}

	return records, nil
}
