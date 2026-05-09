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

type RecurringStream struct {
	ID            string  `json:"id"`
	Frequency     string  `json:"frequency"`
	Amount        float64 `json:"amount"`
	IsApproximate bool    `json:"is_approximate"`
	MerchantName  string  `json:"merchant_name"`
}

type RecurringItem struct {
	Stream        RecurringStream `json:"stream"`
	Date          string          `json:"date"`
	IsPast        bool            `json:"is_past"`
	TransactionID string          `json:"transaction_id"`
	Amount        float64         `json:"amount"`
	AmountDiff    float64         `json:"amount_diff"`
	CategoryName  string          `json:"category_name"`
	AccountID     string          `json:"account_id"`
	AccountName   string          `json:"account_name"`
}

func (s *Service) ListRecurring(ctx context.Context, startDate, endDate string) ([]RecurringTransaction, error) {
	var resp struct {
		RecurringTransactionItems []struct {
			Stream struct {
				ID            string  `json:"id"`
				Frequency     string  `json:"frequency"`
				Amount        float64 `json:"amount"`
				IsApproximate bool    `json:"isApproximate"`
				Merchant      struct {
					ID      string `json:"id"`
					Name    string `json:"name"`
					LogoURL string `json:"logoUrl"`
				} `json:"merchant"`
			} `json:"stream"`
			Date          string  `json:"date"`
			IsPast        bool    `json:"isPast"`
			TransactionID string  `json:"transactionId"`
			Amount        float64 `json:"amount"`
			AmountDiff    float64 `json:"amountDiff"`
			Category      struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"category"`
			Account struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"account"`
		} `json:"recurringTransactionItems"`
	}

	variables := map[string]interface{}{
		"startDate": startDate,
		"endDate":   endDate,
		"filters":   map[string]interface{}{},
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_GetUpcomingRecurringTransactionItems",
		Query:         GetRecurringQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	recurring := make([]RecurringTransaction, len(resp.RecurringTransactionItems))
	for i, r := range resp.RecurringTransactionItems {
		recurring[i] = RecurringTransaction{
			ID:        r.Stream.ID,
			Merchant:  r.Stream.Merchant.Name,
			Amount:    r.Amount,
			Frequency: r.Stream.Frequency,
			NextDate:  r.Date,
			Status:    "active",
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
