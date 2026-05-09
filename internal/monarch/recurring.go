package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetRecurringQuery = queries.Get("recurring/list.graphql")
var UpdateRecurringMutation = queries.Get("recurring/update.graphql")

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

func (s *Service) UpdateRecurring(ctx context.Context, id string, amount float64) (*RecurringTransaction, error) {
	var resp struct {
		UpdateRecurringTransaction struct {
			RecurringTransaction struct {
				ID     string  `json:"id"`
				Amount float64 `json:"amount"`
			} `json:"recurringTransaction"`
		} `json:"updateRecurringTransaction"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "UpdateRecurringTransaction",
		Query:         UpdateRecurringMutation,
		Variables: map[string]interface{}{
			"id":     id,
			"amount": amount,
		},
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &RecurringTransaction{
		ID:     resp.UpdateRecurringTransaction.RecurringTransaction.ID,
		Amount: resp.UpdateRecurringTransaction.RecurringTransaction.Amount,
	}, nil
}
