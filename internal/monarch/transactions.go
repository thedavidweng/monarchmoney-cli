package monarch

import (
	"context"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
	"github.com/monarchmoney-cli/monarch/queries"
)

var GetTransactionsQuery = queries.Get("transactions/list.graphql")
var GetTransactionQuery = queries.Get("transactions/show.graphql")
var GetTransactionsSummaryQuery = queries.Get("transactions/summary.graphql")
var GetDuplicateTransactionsQuery = queries.Get("transactions/duplicates.graphql")
var GetTransactionSplitsQuery = queries.Get("transactions/splits.graphql")
var UpdateTransactionMutation = queries.Get("transactions/update.graphql")
var DeleteTransactionMutation = queries.Get("transactions/delete.graphql")
var UpdateTransactionSplitsMutation = queries.Get("transactions/update_splits.graphql")
var CreateTransactionMutation = queries.Get("transactions/create.graphql")
var SetTransactionTagsMutation = queries.Get("transactions/set_tags.graphql")

type Transaction struct {
	ID       string  `json:"id"`
	Date     string  `json:"date"`
	Amount   float64 `json:"amount"`
	Merchant string  `json:"merchant"`
	Category string  `json:"category"`
	Notes    string  `json:"notes"`
	Tags     []Tag   `json:"tags"`
}

type TransactionSplit struct {
	ID       string  `json:"id"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	Notes    string  `json:"notes"`
}

type SplitInput struct {
	Amount     float64 `json:"amount"`
	CategoryID string  `json:"category_id"`
	Notes      string  `json:"notes"`
}

func (s *Service) GetTransaction(ctx context.Context, id string) (*Transaction, error) {
	var resp struct {
		Transaction struct {
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
			Tags  []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"tags"`
		} `json:"transaction"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetTransaction",
		Query:         GetTransactionQuery,
		Variables:     map[string]interface{}{"id": id},
	}, &resp)

	if err != nil {
		return nil, err
	}

	tags := make([]Tag, len(resp.Transaction.Tags))
	for i, t := range resp.Transaction.Tags {
		tags[i] = Tag{ID: t.ID, Name: t.Name}
	}

	return &Transaction{
		ID:       resp.Transaction.ID,
		Date:     resp.Transaction.Date,
		Amount:   resp.Transaction.Amount,
		Merchant: resp.Transaction.Merchant.Name,
		Category: resp.Transaction.Category.Name,
		Notes:    resp.Transaction.Notes,
		Tags:     tags,
	}, nil
}

func (s *Service) GetTransactionsSummary(ctx context.Context, startDate, endDate string) (map[string]interface{}, error) {
	var resp struct {
		TransactionSummary struct {
			CategorySummaries []struct {
				Category struct {
					Name string `json:"name"`
				} `json:"category"`
				Amount float64 `json:"amount"`
			} `json:"categorySummaries"`
			MerchantSummaries []struct {
				Merchant struct {
					Name string `json:"name"`
				} `json:"merchant"`
				Amount float64 `json:"amount"`
			} `json:"merchantSummaries"`
		} `json:"transactionSummary"`
	}

	variables := make(map[string]interface{})
	if startDate != "" {
		variables["startDate"] = startDate
	}
	if endDate != "" {
		variables["endDate"] = endDate
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetTransactionsSummary",
		Query:         GetTransactionsSummaryQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"categories": resp.TransactionSummary.CategorySummaries,
		"merchants":  resp.TransactionSummary.MerchantSummaries,
	}, nil
}

func (s *Service) GetDuplicateTransactions(ctx context.Context) ([]Transaction, error) {
	var resp struct {
		DuplicateTransactions []struct {
			ID       string  `json:"id"`
			Date     string  `json:"date"`
			Amount   float64 `json:"amount"`
			Merchant struct {
				Name string `json:"name"`
			} `json:"merchant"`
		} `json:"duplicateTransactions"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetDuplicateTransactions",
		Query:         GetDuplicateTransactionsQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	txs := make([]Transaction, len(resp.DuplicateTransactions))
	for i, r := range resp.DuplicateTransactions {
		txs[i] = Transaction{
			ID:       r.ID,
			Date:     r.Date,
			Amount:   r.Amount,
			Merchant: r.Merchant.Name,
		}
	}

	return txs, nil
}

func (s *Service) GetTransactionSplits(ctx context.Context, txID string) ([]TransactionSplit, error) {
	var resp struct {
		Transaction struct {
			Splits []struct {
				ID     string  `json:"id"`
				Amount float64 `json:"amount"`
				Category struct {
					Name string `json:"name"`
				} `json:"category"`
				Notes string `json:"notes"`
			} `json:"splits"`
		} `json:"transaction"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetTransactionSplits",
		Query:         GetTransactionSplitsQuery,
		Variables:     map[string]interface{}{"id": txID},
	}, &resp)

	if err != nil {
		return nil, err
	}

	splits := make([]TransactionSplit, len(resp.Transaction.Splits))
	for i, r := range resp.Transaction.Splits {
		splits[i] = TransactionSplit{
			ID:       r.ID,
			Amount:   r.Amount,
			Category: r.Category.Name,
			Notes:    r.Notes,
		}
	}

	return splits, nil
}

func (s *Service) UpdateTransaction(ctx context.Context, id string, notes *string, categoryID *string) (*Transaction, error) {
	var resp struct {
		UpdateTransaction struct {
			Transaction struct {
				ID       string `json:"id"`
				Notes    string `json:"notes"`
				Category struct {
					Name string `json:"name"`
				} `json:"category"`
			} `json:"transaction"`
		} `json:"updateTransaction"`
	}

	variables := map[string]interface{}{"id": id}
	if notes != nil {
		variables["notes"] = *notes
	}
	if categoryID != nil {
		variables["categoryId"] = *categoryID
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "UpdateTransaction",
		Query:         UpdateTransactionMutation,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &Transaction{
		ID:       resp.UpdateTransaction.Transaction.ID,
		Notes:    resp.UpdateTransaction.Transaction.Notes,
		Category: resp.UpdateTransaction.Transaction.Category.Name,
	}, nil
}

func (s *Service) DeleteTransaction(ctx context.Context, id string) error {
	var resp struct {
		DeleteTransaction struct {
			OK bool `json:"ok"`
		} `json:"deleteTransaction"`
	}

	return s.Client.Do(ctx, &graphql.Request{
		OperationName: "DeleteTransaction",
		Query:         DeleteTransactionMutation,
		Variables:     map[string]interface{}{"id": id},
	}, &resp)
}

func (s *Service) UpdateTransactionSplits(ctx context.Context, txID string, splits []SplitInput) error {
	var resp struct {
		UpdateTransactionSplits struct {
			Transaction struct {
				ID string `json:"id"`
			} `json:"transaction"`
		} `json:"updateTransactionSplits"`
	}

	variables := map[string]interface{}{
		"txId":   txID,
		"splits": splits,
	}

	return s.Client.Do(ctx, &graphql.Request{
		OperationName: "UpdateTransactionSplits",
		Query:         UpdateTransactionSplitsMutation,
		Variables:     variables,
	}, &resp)
}

func (s *Service) CreateTransaction(ctx context.Context, amount float64, merchantName, date, categoryID, accountID, notes string) (*Transaction, error) {
	var resp struct {
		CreateTransaction struct {
			Transaction struct {
				ID     string  `json:"id"`
				Amount float64 `json:"amount"`
				Date   string  `json:"date"`
				Merchant struct {
					Name string `json:"name"`
				} `json:"merchant"`
			} `json:"transaction"`
		} `json:"createTransaction"`
	}

	variables := map[string]interface{}{
		"amount":       amount,
		"merchantName": merchantName,
		"date":         date,
		"categoryId":   categoryID,
	}
	if accountID != "" {
		variables["accountId"] = accountID
	}
	if notes != "" {
		variables["notes"] = notes
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "CreateTransaction",
		Query:         CreateTransactionMutation,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &Transaction{
		ID:       resp.CreateTransaction.Transaction.ID,
		Amount:   resp.CreateTransaction.Transaction.Amount,
		Date:     resp.CreateTransaction.Transaction.Date,
		Merchant: resp.CreateTransaction.Transaction.Merchant.Name,
	}, nil
}

func (s *Service) SetTransactionTags(ctx context.Context, txID string, tagIDs []string) error {
	var resp struct {
		SetTransactionTags struct {
			OK bool `json:"ok"`
		} `json:"setTransactionTags"`
	}

	return s.Client.Do(ctx, &graphql.Request{
		OperationName: "SetTransactionTags",
		Query:         SetTransactionTagsMutation,
		Variables:     map[string]interface{}{"txId": txID, "tagIds": tagIDs},
	}, &resp)
}

type ListTransactionsOptions struct {
	Limit     int
	Offset    int
	Search    string
	StartDate string
	EndDate   string
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

	variables := map[string]interface{}{
		"limit":  opts.Limit,
		"offset": opts.Offset,
	}
	if opts.Search != "" {
		variables["search"] = opts.Search
	}
	if opts.StartDate != "" {
		variables["startDate"] = opts.StartDate
	}
	if opts.EndDate != "" {
		variables["endDate"] = opts.EndDate
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetTransactions",
		Query:         GetTransactionsQuery,
		Variables:     variables,
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
