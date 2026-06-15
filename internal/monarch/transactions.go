package monarch

import (
	"context"
	"strconv"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetTransactionsQuery = queries.Get("transactions/list.graphql")
var GetTransactionQuery = queries.Get("transactions/show.graphql")
var GetTransactionsSummaryQuery = queries.Get("transactions/summary.graphql")
var UpdateTransactionMutation = queries.Get("transactions/update.graphql")
var DeleteTransactionMutation = queries.Get("transactions/delete.graphql")
var CreateTransactionMutation = queries.Get("transactions/create.graphql")
var SetTransactionTagsMutation = queries.Get("transactions/set_tags.graphql")
var GetTransactionSplitsQuery = queries.Get("transactions/get_splits.graphql")
var UpdateTransactionSplitsMutation = queries.Get("transactions/update_splits.graphql")

type Transaction struct {
	ID                      string                   `json:"id"`
	Date                    string                   `json:"date"`
	Amount                  float64                  `json:"amount"`
	Merchant                string                   `json:"merchant"`
	Category                string                   `json:"category"`
	CategoryGroup           TransactionCategoryGroup `json:"category_group,omitempty"`
	Notes                   string                   `json:"notes"`
	Tags                    []Tag                    `json:"tags"`
	Goal                    TransactionGoal          `json:"goal,omitempty"`
	Pending                 bool                     `json:"pending"`
	HideFromReports         bool                     `json:"hide_from_reports"`
	PlaidName               string                   `json:"plaid_name"`
	DataProviderDescription string                   `json:"data_provider_description"`
	IsRecurring             bool                     `json:"is_recurring"`
	ReviewStatus            string                   `json:"review_status"`
	NeedsReview             bool                     `json:"needs_review"`
	IsSplitTransaction      bool                     `json:"is_split_transaction"`
	CreatedAt               string                   `json:"created_at"`
	UpdatedAt               string                   `json:"updated_at"`
	AccountID               string                   `json:"account_id"`
	AccountOrder            int                      `json:"account_order"`
	AccountTypeGroup        string                   `json:"account_type_group"`
	OwnerDisplayName        string                   `json:"owner_display_name"`
}

type TransactionCategoryGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type TransactionGoal struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TransactionSplit struct {
	ID       string  `json:"id"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	Merchant string  `json:"merchant"`
	Notes    string  `json:"notes"`
}

type SplitInput struct {
	Amount       float64 `json:"amount"`
	CategoryID   string  `json:"category_id"`
	MerchantName string  `json:"merchant_name"`
	Notes        string  `json:"notes"`
}

func (s *Service) GetTransaction(ctx context.Context, id string) (*Transaction, error) {
	var resp struct {
		GetTransaction struct {
			ID       string  `json:"id"`
			Date     string  `json:"date"`
			Amount   float64 `json:"amount"`
			Merchant struct {
				Name string `json:"name"`
			} `json:"merchant"`
			Category struct {
				Name string `json:"name"`
			} `json:"category"`
			Notes              string `json:"notes"`
			Pending            bool   `json:"pending"`
			HideFromReports    bool   `json:"hideFromReports"`
			PlaidName          string `json:"plaidName"`
			IsRecurring        bool   `json:"isRecurring"`
			ReviewStatus       string `json:"reviewStatus"`
			NeedsReview        bool   `json:"needsReview"`
			IsSplitTransaction bool   `json:"isSplitTransaction"`
			CreatedAt          string `json:"createdAt"`
			UpdatedAt          string `json:"updatedAt"`
			Account            struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"account"`
			Tags []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Color string `json:"color"`
				Order int    `json:"order"`
			} `json:"tags"`
		} `json:"getTransaction"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetTransaction",
		Query:         GetTransactionQuery,
		Variables:     map[string]any{"id": id},
	}, &resp)

	if err != nil {
		return nil, err
	}

	tags := make([]Tag, len(resp.GetTransaction.Tags))
	for i, t := range resp.GetTransaction.Tags {
		tags[i] = Tag{ID: t.ID, Name: t.Name, Color: t.Color}
	}

	return &Transaction{
		ID:                 resp.GetTransaction.ID,
		Date:               resp.GetTransaction.Date,
		Amount:             resp.GetTransaction.Amount,
		Merchant:           resp.GetTransaction.Merchant.Name,
		Category:           resp.GetTransaction.Category.Name,
		Notes:              resp.GetTransaction.Notes,
		Pending:            resp.GetTransaction.Pending,
		HideFromReports:    resp.GetTransaction.HideFromReports,
		PlaidName:          resp.GetTransaction.PlaidName,
		IsRecurring:        resp.GetTransaction.IsRecurring,
		ReviewStatus:       resp.GetTransaction.ReviewStatus,
		NeedsReview:        resp.GetTransaction.NeedsReview,
		IsSplitTransaction: resp.GetTransaction.IsSplitTransaction,
		CreatedAt:          resp.GetTransaction.CreatedAt,
		UpdatedAt:          resp.GetTransaction.UpdatedAt,
		AccountID:          resp.GetTransaction.Account.ID,
		Tags:               tags,
	}, nil
}

type TransactionSummaryResult struct {
	Avg        float64 `json:"avg"`
	Count      int     `json:"count"`
	Max        float64 `json:"max"`
	MaxExpense float64 `json:"max_expense"`
	Sum        float64 `json:"sum"`
	SumIncome  float64 `json:"sum_income"`
	SumExpense float64 `json:"sum_expense"`
	First      string  `json:"first"`
	Last       string  `json:"last"`
}

func (s *Service) GetTransactionsSummary(ctx context.Context, startDate, endDate string) (*TransactionSummaryResult, error) {
	var resp struct {
		Aggregates []struct {
			Summary struct {
				Avg        float64 `json:"avg"`
				Count      int     `json:"count"`
				Max        float64 `json:"max"`
				MaxExpense float64 `json:"maxExpense"`
				Sum        float64 `json:"sum"`
				SumIncome  float64 `json:"sumIncome"`
				SumExpense float64 `json:"sumExpense"`
				First      string  `json:"first"`
				Last       string  `json:"last"`
			} `json:"summary"`
		} `json:"aggregates"`
	}

	filters := map[string]any{
		"search":     "",
		"categories": []string{},
		"accounts":   []string{},
		"tags":       []string{},
	}
	if startDate != "" {
		filters["startDate"] = startDate
	}
	if endDate != "" {
		filters["endDate"] = endDate
	}

	variables := map[string]any{
		"filters": filters,
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetTransactionsPage",
		Query:         GetTransactionsSummaryQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	if len(resp.Aggregates) == 0 {
		return &TransactionSummaryResult{}, nil
	}

	return &TransactionSummaryResult{
		Avg:        resp.Aggregates[0].Summary.Avg,
		Count:      resp.Aggregates[0].Summary.Count,
		Max:        resp.Aggregates[0].Summary.Max,
		MaxExpense: resp.Aggregates[0].Summary.MaxExpense,
		Sum:        resp.Aggregates[0].Summary.Sum,
		SumIncome:  resp.Aggregates[0].Summary.SumIncome,
		SumExpense: resp.Aggregates[0].Summary.SumExpense,
		First:      resp.Aggregates[0].Summary.First,
		Last:       resp.Aggregates[0].Summary.Last,
	}, nil
}

func (s *Service) GetDuplicateTransactions(ctx context.Context, startDate, endDate string) ([]Transaction, error) {
	const pageSize = 1000
	all := make([]Transaction, 0, pageSize)
	for offset := 0; ; offset += pageSize {
		page, total, err := s.ListTransactions(ctx, ListTransactionsOptions{
			Limit:     pageSize,
			Offset:    offset,
			StartDate: startDate,
			EndDate:   endDate,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) == 0 || offset+len(page) >= total {
			break
		}
	}

	counts := make(map[string]int, len(all))
	for _, tx := range all {
		counts[duplicateTransactionKey(tx)]++
	}

	duplicates := make([]Transaction, 0)
	for _, tx := range all {
		if counts[duplicateTransactionKey(tx)] > 1 {
			duplicates = append(duplicates, tx)
		}
	}

	return duplicates, nil
}

func (s *Service) GetTransactionSplits(ctx context.Context, txID string) ([]TransactionSplit, error) {
	var resp struct {
		GetTransaction struct {
			ID                string  `json:"id"`
			Amount            float64 `json:"amount"`
			SplitTransactions []struct {
				ID       string  `json:"id"`
				Amount   float64 `json:"amount"`
				Notes    string  `json:"notes"`
				Merchant struct {
					Name string `json:"name"`
				} `json:"merchant"`
				Category struct {
					Name string `json:"name"`
				} `json:"category"`
			} `json:"splitTransactions"`
		} `json:"getTransaction"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "TransactionSplitQuery",
		Query:         GetTransactionSplitsQuery,
		Variables:     map[string]any{"id": txID},
	}, &resp)
	if err != nil {
		return nil, err
	}

	splits := make([]TransactionSplit, len(resp.GetTransaction.SplitTransactions))
	for i, s := range resp.GetTransaction.SplitTransactions {
		splits[i] = TransactionSplit{
			ID:       s.ID,
			Amount:   s.Amount,
			Category: s.Category.Name,
			Merchant: s.Merchant.Name,
			Notes:    s.Notes,
		}
	}
	return splits, nil
}

func (s *Service) UpdateTransaction(ctx context.Context, id string, notes *string, categoryID *string, amount *float64, date *string, merchantName *string, hideFromReports *bool, needsReview *bool) (*Transaction, error) {
	var resp struct {
		UpdateTransaction struct {
			Transaction struct {
				ID              string  `json:"id"`
				Amount          float64 `json:"amount"`
				Date            string  `json:"date"`
				Notes           string  `json:"notes"`
				HideFromReports bool    `json:"hideFromReports"`
				NeedsReview     bool    `json:"needsReview"`
				Category        struct {
					Name string `json:"name"`
				} `json:"category"`
				Merchant struct {
					Name string `json:"name"`
				} `json:"merchant"`
			} `json:"transaction"`
		} `json:"updateTransaction"`
	}

	variables := map[string]any{
		"input": map[string]any{
			"id": id,
		},
	}
	input := variables["input"].(map[string]any)
	if notes != nil {
		input["notes"] = *notes
	}
	if categoryID != nil {
		input["category"] = *categoryID
	}
	if amount != nil {
		input["amount"] = *amount
	}
	if date != nil {
		input["date"] = *date
	}
	if merchantName != nil {
		input["name"] = *merchantName
	}
	if hideFromReports != nil {
		input["hideFromReports"] = *hideFromReports
	}
	if needsReview != nil {
		input["needsReview"] = *needsReview
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_TransactionDrawerUpdateTransaction",
		Query:         UpdateTransactionMutation,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &Transaction{
		ID:              resp.UpdateTransaction.Transaction.ID,
		Amount:          resp.UpdateTransaction.Transaction.Amount,
		Date:            resp.UpdateTransaction.Transaction.Date,
		Notes:           resp.UpdateTransaction.Transaction.Notes,
		Category:        resp.UpdateTransaction.Transaction.Category.Name,
		Merchant:        resp.UpdateTransaction.Transaction.Merchant.Name,
		HideFromReports: resp.UpdateTransaction.Transaction.HideFromReports,
		NeedsReview:     resp.UpdateTransaction.Transaction.NeedsReview,
	}, nil
}

func (s *Service) DeleteTransaction(ctx context.Context, id string) error {
	var resp struct {
		DeleteTransaction struct {
			Deleted bool `json:"deleted"`
		} `json:"deleteTransaction"`
	}

	return s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_DeleteTransactionMutation",
		Query:         DeleteTransactionMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"transactionId": id,
			},
		},
	}, &resp)
}

func (s *Service) UpdateTransactionSplits(ctx context.Context, txID string, splits []SplitInput) error {
	var resp struct {
		UpdateTransactionSplit struct {
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
			Transaction struct {
				ID string `json:"id"`
			} `json:"transaction"`
		} `json:"updateTransactionSplit"`
	}

	splitData := make([]map[string]any, len(splits))
	for i, s := range splits {
		sd := map[string]any{
			"amount": s.Amount,
		}
		if s.CategoryID != "" {
			sd["categoryId"] = s.CategoryID
		}
		if s.MerchantName != "" {
			sd["merchantName"] = s.MerchantName
		}
		if s.Notes != "" {
			sd["notes"] = s.Notes
		}
		splitData[i] = sd
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_SplitTransactionMutation",
		Query:         UpdateTransactionSplitsMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"transactionId": txID,
				"splitData":     splitData,
			},
		},
	}, &resp)
	if err != nil {
		return err
	}

	if len(resp.UpdateTransactionSplit.Errors) > 0 {
		return errors.New(errors.APIError, resp.UpdateTransactionSplit.Errors[0].Message, errors.CatAPI, false, nil)
	}
	return nil
}

func (s *Service) CreateTransaction(ctx context.Context, amount float64, merchantName, date, categoryID, accountID, notes string) (*Transaction, error) {
	var resp struct {
		CreateTransaction struct {
			Transaction struct {
				ID       string  `json:"id"`
				Amount   float64 `json:"amount"`
				Date     string  `json:"date"`
				Merchant struct {
					Name string `json:"name"`
				} `json:"merchant"`
			} `json:"transaction"`
		} `json:"createTransaction"`
	}

	variables := map[string]any{
		"input": map[string]any{
			"date":                date,
			"accountId":           accountID,
			"amount":              amount,
			"merchantName":        merchantName,
			"categoryId":          categoryID,
			"notes":               notes,
			"shouldUpdateBalance": false,
		},
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_CreateTransactionMutation",
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
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"setTransactionTags"`
	}

	return s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_SetTransactionTags",
		Query:         SetTransactionTagsMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"transactionId": txID,
				"tagIds":        tagIDs,
			},
		},
	}, &resp)
}

type ListTransactionsOptions struct {
	Limit           int
	Offset          int
	Search          string
	StartDate       string
	EndDate         string
	CategoryIDs     []string
	AccountIDs      []string
	TagIDs          []string
	GoalIDs         []string
	NeedsReview     *bool
	HasNotes        *bool
	IsSplit         *bool
	IsRecurring     *bool
	Pending         *bool
	HideFromReports *bool
}

func normalizeListTransactionsOptions(opts ListTransactionsOptions) ListTransactionsOptions {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	return opts
}

func buildListTransactionsFilters(opts ListTransactionsOptions) map[string]any {
	filters := map[string]any{
		"search":     opts.Search,
		"categories": nonNilStrings(opts.CategoryIDs),
		"accounts":   nonNilStrings(opts.AccountIDs),
		"tags":       nonNilStrings(opts.TagIDs),
	}

	if opts.StartDate != "" {
		filters["startDate"] = opts.StartDate
	}
	if opts.EndDate != "" {
		filters["endDate"] = opts.EndDate
	}

	addOptionalBoolFilter(filters, "needsReview", opts.NeedsReview)
	addOptionalBoolFilter(filters, "hasNotes", opts.HasNotes)
	addOptionalBoolFilter(filters, "isSplit", opts.IsSplit)
	addOptionalBoolFilter(filters, "isRecurring", opts.IsRecurring)
	addOptionalBoolFilter(filters, "isPending", opts.Pending)
	addOptionalBoolFilter(filters, "hideFromReports", opts.HideFromReports)

	if len(opts.GoalIDs) > 0 {
		filters["goals"] = opts.GoalIDs
	}

	return filters
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func addOptionalBoolFilter(filters map[string]any, key string, value *bool) {
	if value != nil {
		filters[key] = *value
	}
}

func (s *Service) ListTransactions(ctx context.Context, opts ListTransactionsOptions) ([]Transaction, int, error) {
	var resp struct {
		AllTransactions struct {
			Results []struct {
				ID                      string  `json:"id"`
				Date                    string  `json:"date"`
				Amount                  float64 `json:"amount"`
				Pending                 bool    `json:"pending"`
				HideFromReports         bool    `json:"hideFromReports"`
				DataProviderDescription string  `json:"dataProviderDescription"`
				PlaidName               string  `json:"plaidName"`
				Notes                   string  `json:"notes"`
				IsRecurring             bool    `json:"isRecurring"`
				ReviewStatus            string  `json:"reviewStatus"`
				NeedsReview             bool    `json:"needsReview"`
				IsSplitTransaction      bool    `json:"isSplitTransaction"`
				CreatedAt               string  `json:"createdAt"`
				UpdatedAt               string  `json:"updatedAt"`
				Category                struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Group struct {
						ID   string `json:"id"`
						Name string `json:"name"`
						Type string `json:"type"`
					} `json:"group"`
				} `json:"category"`
				Merchant struct {
					Name string `json:"name"`
					ID   string `json:"id"`
				} `json:"merchant"`
				Account struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
					Order       int    `json:"order"`
					Type        struct {
						Group string `json:"group"`
					} `json:"type"`
				} `json:"account"`
				OwnedByUser struct {
					DisplayName string `json:"displayName"`
				} `json:"ownedByUser"`
				Goal struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"goal"`
				Tags []struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Color string `json:"color"`
					Order int    `json:"order"`
				} `json:"tags"`
			} `json:"results"`
			TotalCount int `json:"totalCount"`
		} `json:"allTransactions"`
	}

	opts = normalizeListTransactionsOptions(opts)
	filters := buildListTransactionsFilters(opts)

	variables := map[string]any{
		"offset":  opts.Offset,
		"limit":   opts.Limit,
		"filters": filters,
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetTransactionsList",
		Query:         GetTransactionsQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, 0, err
	}

	txs := make([]Transaction, len(resp.AllTransactions.Results))
	for i, r := range resp.AllTransactions.Results {
		tags := make([]Tag, len(r.Tags))
		for j, t := range r.Tags {
			tags[j] = Tag{ID: t.ID, Name: t.Name, Color: t.Color}
		}
		txs[i] = Transaction{
			ID:       r.ID,
			Date:     r.Date,
			Amount:   r.Amount,
			Merchant: r.Merchant.Name,
			Category: r.Category.Name,
			CategoryGroup: TransactionCategoryGroup{
				ID:   r.Category.Group.ID,
				Name: r.Category.Group.Name,
				Type: r.Category.Group.Type,
			},
			Notes: r.Notes,
			Tags:  tags,
			Goal: TransactionGoal{
				ID:   r.Goal.ID,
				Name: r.Goal.Name,
			},
			Pending:                 r.Pending,
			HideFromReports:         r.HideFromReports,
			PlaidName:               r.PlaidName,
			DataProviderDescription: r.DataProviderDescription,
			IsRecurring:             r.IsRecurring,
			ReviewStatus:            r.ReviewStatus,
			NeedsReview:             r.NeedsReview,
			IsSplitTransaction:      r.IsSplitTransaction,
			CreatedAt:               r.CreatedAt,
			UpdatedAt:               r.UpdatedAt,
			AccountID:               r.Account.ID,
			AccountOrder:            r.Account.Order,
			AccountTypeGroup:        r.Account.Type.Group,
			OwnerDisplayName:        r.OwnedByUser.DisplayName,
		}
	}

	return txs, resp.AllTransactions.TotalCount, nil
}

func (s *Service) ListAllTransactions(ctx context.Context, opts ListTransactionsOptions) ([]Transaction, error) {
	if opts.Limit <= 0 {
		opts.Limit = 1000
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	all := make([]Transaction, 0, opts.Limit)
	for {
		page, total, err := s.ListTransactions(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) == 0 || opts.Offset+len(page) >= total {
			break
		}
		opts.Offset += len(page)
	}
	return all, nil
}

func duplicateTransactionKey(tx Transaction) string {
	return tx.Date + "|" + strconv.FormatFloat(tx.Amount, 'f', 2, 64) + "|" + tx.PlaidName + "|" + tx.AccountID
}
