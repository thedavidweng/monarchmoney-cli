package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var (
	GetRecurringQuery             = queries.Get("recurring/list.graphql")
	UpdateRecurringMutation       = queries.Get("recurring/update.graphql")
	ListRecurringStreamsQuery     = queries.Get("recurring/streams.graphql")
	GetRecurringSummaryQuery      = queries.Get("recurring/summary.graphql")
	SetMerchantRecurrenceMutation = queries.Get("recurring/set_recurrence.graphql")
	RemoveRecurringStreamMutation = queries.Get("recurring/remove.graphql")
	ReviewRecurringStreamMutation = queries.Get("recurring/review.graphql")
)

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
	items, err := s.ListRecurringItems(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	recurring := make([]RecurringTransaction, len(items))
	for i := range items {
		r := &items[i]
		recurring[i] = RecurringTransaction{
			ID:        r.Stream.ID,
			Merchant:  r.Stream.MerchantName,
			Amount:    r.Amount,
			Frequency: r.Stream.Frequency,
			NextDate:  r.Date,
			Status:    "active",
		}
	}

	return recurring, nil
}

func (s *Service) ListRecurringItems(ctx context.Context, startDate, endDate string) ([]RecurringItem, error) {
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

	variables := map[string]any{
		"startDate": startDate,
		"endDate":   endDate,
		"filters":   map[string]any{},
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_GetUpcomingRecurringTransactionItems",
		Query:         GetRecurringQuery,
		Variables:     variables,
	}, &resp)
	if err != nil {
		return nil, err
	}

	recurring := make([]RecurringItem, len(resp.RecurringTransactionItems))
	for i := range resp.RecurringTransactionItems {
		r := &resp.RecurringTransactionItems[i]
		recurring[i] = RecurringItem{
			Stream: RecurringStream{
				ID:            r.Stream.ID,
				Frequency:     r.Stream.Frequency,
				Amount:        r.Stream.Amount,
				IsApproximate: r.Stream.IsApproximate,
				MerchantName:  r.Stream.Merchant.Name,
			},
			Date:          r.Date,
			IsPast:        r.IsPast,
			TransactionID: r.TransactionID,
			Amount:        r.Amount,
			AmountDiff:    r.AmountDiff,
			CategoryName:  r.Category.Name,
			AccountID:     r.Account.ID,
			AccountName:   r.Account.DisplayName,
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

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "UpdateRecurringTransaction",
		Query:         UpdateRecurringMutation,
		Variables: map[string]any{
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

type RecurringStreamDetail struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Frequency     string  `json:"frequency"`
	Amount        float64 `json:"amount"`
	BaseDate      string  `json:"base_date,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
	IsApproximate bool    `json:"is_approximate"`
	MerchantID    string  `json:"merchant_id,omitempty"`
	MerchantName  string  `json:"merchant_name"`
	Category      string  `json:"category,omitempty"`
	Account       string  `json:"account,omitempty"`
	NextDate      string  `json:"next_date,omitempty"`
	NextAmount    float64 `json:"next_amount,omitempty"`
}

type RecurringSummary struct {
	ExpenseCompleted float64 `json:"expense_completed"`
	ExpenseRemaining float64 `json:"expense_remaining"`
	ExpenseTotal     float64 `json:"expense_total"`
	ExpenseCount     int     `json:"expense_count"`
	IncomeCompleted  float64 `json:"income_completed"`
	IncomeRemaining  float64 `json:"income_remaining"`
	IncomeTotal      float64 `json:"income_total"`
}

type rawRecurringStream struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Frequency     string  `json:"frequency"`
	Amount        float64 `json:"amount"`
	BaseDate      string  `json:"baseDate"`
	IsActive      *bool   `json:"isActive"`
	IsApproximate bool    `json:"isApproximate"`
	Merchant      *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"merchant"`
}

func toRecurringStreamDetail(stream *rawRecurringStream, category, account, nextDate string, nextAmount float64) *RecurringStreamDetail {
	d := &RecurringStreamDetail{
		ID: stream.ID, Name: stream.Name, Frequency: stream.Frequency,
		Amount: stream.Amount, BaseDate: stream.BaseDate, IsActive: stream.IsActive,
		IsApproximate: stream.IsApproximate, Category: category, Account: account,
		NextDate: nextDate, NextAmount: nextAmount,
	}
	if stream.Merchant != nil {
		d.MerchantID = stream.Merchant.ID
		d.MerchantName = stream.Merchant.Name
	}
	return d
}

func (s *Service) ListRecurringStreams(ctx context.Context) ([]*RecurringStreamDetail, error) {
	var resp struct {
		RecurringTransactionStreams []struct {
			Stream *rawRecurringStream `json:"stream"`
			Next   *struct {
				Date   string  `json:"date"`
				Amount float64 `json:"amount"`
			} `json:"nextForecastedTransaction"`
			Category *struct {
				Name string `json:"name"`
			} `json:"category"`
			Account *struct {
				DisplayName string `json:"displayName"`
			} `json:"account"`
		} `json:"recurringTransactionStreams"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_GetAllRecurringTransactionItems",
		Query:         ListRecurringStreamsQuery,
		Variables: map[string]any{
			"filters":            map[string]any{},
			"includePending":     true,
			"includeLiabilities": true,
		},
	}, &resp)
	if err != nil {
		return nil, err
	}

	out := make([]*RecurringStreamDetail, 0, len(resp.RecurringTransactionStreams))
	for _, item := range resp.RecurringTransactionStreams {
		if item.Stream == nil {
			continue
		}
		category, account, nextDate, nextAmount := "", "", "", 0.0
		if item.Category != nil {
			category = item.Category.Name
		}
		if item.Account != nil {
			account = item.Account.DisplayName
		}
		if item.Next != nil {
			nextDate, nextAmount = item.Next.Date, item.Next.Amount
		}
		out = append(out, toRecurringStreamDetail(item.Stream, category, account, nextDate, nextAmount))
	}
	return out, nil
}

func (s *Service) GetRecurringStream(ctx context.Context, id string) (*RecurringStreamDetail, error) {
	streams, err := s.ListRecurringStreams(ctx)
	if err != nil {
		return nil, err
	}
	for _, stream := range streams {
		if stream.ID == id {
			return stream, nil
		}
	}
	return nil, errors.New(errors.ResourceNotFound, "recurring stream not found", errors.CatAPI, false, nil)
}

func (s *Service) GetRecurringSummary(ctx context.Context, startDate, endDate string) (*RecurringSummary, error) {
	var resp struct {
		AggregatedRecurringItems *struct {
			AggregatedSummary *struct {
				Expense *struct {
					Completed float64 `json:"completed"`
					Remaining float64 `json:"remaining"`
					Total     float64 `json:"total"`
					Count     int     `json:"count"`
				} `json:"expense"`
				Income *struct {
					Completed float64 `json:"completed"`
					Remaining float64 `json:"remaining"`
					Total     float64 `json:"total"`
				} `json:"income"`
			} `json:"aggregatedSummary"`
		} `json:"aggregatedRecurringItems"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_GetAggregatedRecurringItems",
		Query:         GetRecurringSummaryQuery,
		Variables: map[string]any{
			"startDate": startDate,
			"endDate":   endDate,
			"filters":   map[string]any{},
		},
	}, &resp)
	if err != nil {
		return nil, err
	}
	summary := &RecurringSummary{}
	if resp.AggregatedRecurringItems != nil && resp.AggregatedRecurringItems.AggregatedSummary != nil {
		agg := resp.AggregatedRecurringItems.AggregatedSummary
		if agg.Expense != nil {
			summary.ExpenseCompleted, summary.ExpenseRemaining = agg.Expense.Completed, agg.Expense.Remaining
			summary.ExpenseTotal, summary.ExpenseCount = agg.Expense.Total, agg.Expense.Count
		}
		if agg.Income != nil {
			summary.IncomeCompleted, summary.IncomeRemaining = agg.Income.Completed, agg.Income.Remaining
			summary.IncomeTotal = agg.Income.Total
		}
	}
	return summary, nil
}

func (s *Service) setMerchantRecurrence(ctx context.Context, merchantID string, recurrence map[string]any) error {
	var resp struct {
		UpdateMerchant struct {
			Errors []payloadError `json:"errors"`
		} `json:"updateMerchant"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_RecurringUpdateMerchant",
		Query:         SetMerchantRecurrenceMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"merchantId": merchantID,
				"recurrence": recurrence,
			},
		},
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.UpdateMerchant.Errors, "failed to update merchant recurrence"); apiErr != nil {
		return apiErr
	}
	return nil
}

type RecurringStreamInput struct {
	Frequency *string
	Amount    *float64
	BaseDate  *string
	IsActive  *bool
}

func (s *Service) CreateRecurringStream(ctx context.Context, merchantID string, input *RecurringStreamInput) (*RecurringStreamDetail, error) {
	recurrence := map[string]any{"isRecurring": true}
	if input.Frequency != nil {
		recurrence["frequency"] = *input.Frequency
	}
	if input.Amount != nil {
		recurrence["amount"] = *input.Amount
	}
	if input.BaseDate != nil {
		recurrence["baseDate"] = *input.BaseDate
	}
	if input.IsActive != nil {
		recurrence["isActive"] = *input.IsActive
	}
	if err := s.setMerchantRecurrence(ctx, merchantID, recurrence); err != nil {
		return nil, err
	}
	streams, err := s.ListRecurringStreams(ctx)
	if err != nil {
		return nil, err
	}
	for _, stream := range streams {
		if stream.MerchantID == merchantID {
			return stream, nil
		}
	}
	return nil, errors.New(errors.APISchemaChanged, "recurring stream not found after creation", errors.CatAPI, false, nil)
}

func (s *Service) UpdateRecurringStream(ctx context.Context, id string, input *RecurringStreamInput) (*RecurringStreamDetail, error) {
	current, err := s.GetRecurringStream(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.MerchantID == "" {
		return nil, errors.New(errors.APIError, "only merchant-backed recurring streams can be updated", errors.CatAPI, false, nil)
	}
	frequency, amount, baseDate, isActive := current.Frequency, current.Amount, current.BaseDate, current.IsActive
	if input.Frequency != nil {
		frequency = *input.Frequency
	}
	if input.Amount != nil {
		amount = *input.Amount
	}
	if input.BaseDate != nil {
		baseDate = *input.BaseDate
	}
	if input.IsActive != nil {
		isActive = input.IsActive
	}
	recurrence := map[string]any{
		"isRecurring": true,
		"frequency":   frequency,
		"amount":      amount,
		"baseDate":    baseDate,
	}
	if isActive != nil {
		recurrence["isActive"] = *isActive
	}
	if err := s.setMerchantRecurrence(ctx, current.MerchantID, recurrence); err != nil {
		return nil, err
	}
	return s.GetRecurringStream(ctx, id)
}

func (s *Service) RemoveRecurringStream(ctx context.Context, id string) error {
	var resp struct {
		MarkStreamAsNotRecurring struct {
			Success bool           `json:"success"`
			Errors  []payloadError `json:"errors"`
		} `json:"markStreamAsNotRecurring"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_MarkAsNotRecurring",
		Query:         RemoveRecurringStreamMutation,
		Variables:     map[string]any{"streamId": id},
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.MarkStreamAsNotRecurring.Errors, "failed to remove recurring stream"); apiErr != nil {
		return apiErr
	}
	if !resp.MarkStreamAsNotRecurring.Success {
		return errors.New(errors.APIError, "failed to remove recurring stream", errors.CatAPI, false, nil)
	}
	return nil
}

type RecurringStreamReview struct {
	StreamID     string `json:"stream_id"`
	ReviewStatus string `json:"review_status"`
}

func (s *Service) ReviewRecurringStream(ctx context.Context, streamID, status string) (*RecurringStreamReview, error) {
	var resp struct {
		ReviewRecurringStream struct {
			Stream *struct {
				ID           string `json:"id"`
				ReviewStatus string `json:"reviewStatus"`
			} `json:"stream"`
			Errors *struct {
				Message string `json:"message"`
				Code    string `json:"code"`
			} `json:"errors"`
		} `json:"reviewRecurringStream"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_ReviewStream",
		Query:         ReviewRecurringStreamMutation,
		Variables:     map[string]any{"input": map[string]any{"streamId": streamID, "reviewStatus": status}},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.ReviewRecurringStream.Errors != nil && resp.ReviewRecurringStream.Errors.Message != "" {
		return nil, errors.New(errors.APIError, resp.ReviewRecurringStream.Errors.Message, errors.CatAPI, false, nil)
	}
	stream := resp.ReviewRecurringStream.Stream
	if stream == nil || stream.ID == "" {
		return nil, errors.New(errors.APISchemaChanged, "stream review response missing stream", errors.CatAPI, false, nil)
	}
	return &RecurringStreamReview{StreamID: stream.ID, ReviewStatus: stream.ReviewStatus}, nil
}
