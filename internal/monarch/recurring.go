package monarch

import (
	"context"
	_ "embed"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

//go:embed queries/recurring/list.graphql
var GetRecurringQuery string

type RecurringTransaction struct {
	ID        string  `json:"id"`
	Merchant  string  `json:"merchant"`
	Amount    float64 `json:"amount"`
	Frequency string  `json:"frequency"`
	NextDate  string  `json:"next_date"`
	Status    string  `json:"status"`
}

func (s *Service) ListRecurring(ctx context.Context) ([]RecurringTransaction, error) {
	var resp struct {
		RecurringTransactions []struct {
			ID       string `json:"id"`
			Merchant struct {
				Name string `json:"name"`
			} `json:"merchant"`
			Amount    float64 `json:"amount"`
			Frequency string  `json:"frequency"`
			NextDate  string  `json:"nextDate"`
			Status    string  `json:"status"`
		} `json:"recurringTransactions"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetRecurringTransactions",
		Query:         GetRecurringQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	recurring := make([]RecurringTransaction, len(resp.RecurringTransactions))
	for i, r := range resp.RecurringTransactions {
		recurring[i] = RecurringTransaction{
			ID:        r.ID,
			Merchant:  r.Merchant.Name,
			Amount:    r.Amount,
			Frequency: r.Frequency,
			NextDate:  r.NextDate,
			Status:    r.Status,
		}
	}

	return recurring, nil
}
