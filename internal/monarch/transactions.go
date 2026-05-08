package monarch

import (
	"context"
	_ "embed"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

//go:embed queries/transactions/list.graphql
var GetTransactionsQuery string

type Transaction struct {
	ID       string  `json:"id"`
	Date     string  `json:"date"`
	Amount   float64 `json:"amount"`
	Merchant string  `json:"merchant"`
	Category string  `json:"category"`
	Notes    string  `json:"notes"`
}

type ListTransactionsOptions struct {
	Limit  int
	Offset int
}

func (s *Service) ListTransactions(ctx context.Context, opts ListTransactionsOptions) ([]Transaction, int, error) {
	var resp struct {
		AllTransactions struct {
			Results []struct {
				ID       string  `json:"id"`
				Date     string  `json:"date"`
				Amount   float64 `json:"amount"`
				Merchant struct {
					Name string `json:"name"`
				} `json:"merchant"`
				Category struct {
					Name string `json:"name"`
				} `json:"category"`
				Notes string `json:"notes"`
			} `json:"results"`
			TotalCount int `json:"totalCount"`
		} `json:"allTransactions"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetTransactions",
		Query:         GetTransactionsQuery,
		Variables: map[string]interface{}{
			"limit":  opts.Limit,
			"offset": opts.Offset,
		},
	}, &resp)

	if err != nil {
		return nil, 0, err
	}

	txs := make([]Transaction, len(resp.AllTransactions.Results))
	for i, r := range resp.AllTransactions.Results {
		txs[i] = Transaction{
			ID:       r.ID,
			Date:     r.Date,
			Amount:   r.Amount,
			Merchant: r.Merchant.Name,
			Category: r.Category.Name,
			Notes:    r.Notes,
		}
	}

	return txs, resp.AllTransactions.TotalCount, nil
}
