package monarch

import (
	"context"
	_ "embed"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

//go:embed queries/cashflow/summary.graphql
var GetCashflowSummaryQuery string

type CashflowSummary struct {
	Income      float64 `json:"income"`
	Expense     float64 `json:"expense"`
	Savings     float64 `json:"savings"`
	SavingsRate float64 `json:"savings_rate"`
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
