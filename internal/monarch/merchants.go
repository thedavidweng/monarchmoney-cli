package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var (
	ListMerchantsQuery     = queries.Get("merchants/list.graphql")
	GetMerchantQuery       = queries.Get("merchants/show.graphql")
	UpdateMerchantMutation = queries.Get("merchants/update.graphql")
	DeleteMerchantMutation = queries.Get("merchants/delete.graphql")
)

type MerchantDefaultCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Merchant struct {
	ID                             string                   `json:"id"`
	Name                           string                   `json:"name"`
	LogoURL                        string                   `json:"logo_url,omitempty"`
	TransactionCount               int                      `json:"transaction_count"`
	RuleCount                      int                      `json:"rule_count,omitempty"`
	CanBeDeleted                   *bool                    `json:"can_be_deleted,omitempty"`
	CreatedAt                      string                   `json:"created_at,omitempty"`
	DefaultCategory                *MerchantDefaultCategory `json:"default_category,omitempty"`
	DefaultCategoryApplicationMode string                   `json:"default_category_application_mode,omitempty"`
	ShowDefaultCategoryPrompt      *bool                    `json:"show_default_category_prompt,omitempty"`
	RecurringStreamID              string                   `json:"recurring_stream_id,omitempty"`
}

type rawMerchant struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	LogoURL          string `json:"logoUrl"`
	TransactionCount int    `json:"transactionCount"`
	RuleCount        int    `json:"ruleCount"`
	CanBeDeleted     *bool  `json:"canBeDeleted"`
	CreatedAt        string `json:"createdAt"`
	DefaultCategory  *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"defaultCategory"`
	DefaultCategoryApplicationMode string `json:"defaultCategoryApplicationMode"`
	ShowDefaultCategoryPrompt      *bool  `json:"showDefaultCategoryPrompt"`
	RecurringTransactionStream     *struct {
		ID string `json:"id"`
	} `json:"recurringTransactionStream"`
}

func toMerchant(r *rawMerchant) *Merchant {
	m := &Merchant{
		ID:                             r.ID,
		Name:                           r.Name,
		LogoURL:                        r.LogoURL,
		TransactionCount:               r.TransactionCount,
		RuleCount:                      r.RuleCount,
		CanBeDeleted:                   r.CanBeDeleted,
		CreatedAt:                      r.CreatedAt,
		DefaultCategoryApplicationMode: r.DefaultCategoryApplicationMode,
		ShowDefaultCategoryPrompt:      r.ShowDefaultCategoryPrompt,
	}
	if r.DefaultCategory != nil {
		m.DefaultCategory = &MerchantDefaultCategory{ID: r.DefaultCategory.ID, Name: r.DefaultCategory.Name}
	}
	if r.RecurringTransactionStream != nil {
		m.RecurringStreamID = r.RecurringTransactionStream.ID
	}
	return m
}

type ListMerchantsOptions struct {
	Search                              string
	Limit                               int
	Offset                              int
	OrderBy                             string
	HasDefaultCategory                  *bool
	IncludeMerchantsWithoutTransactions *bool
	IncludeIDs                          []string
}

func (s *Service) ListMerchants(ctx context.Context, opts *ListMerchantsOptions) ([]*Merchant, error) {
	var resp struct {
		Merchants []*rawMerchant `json:"merchants"`
	}

	variables := map[string]any{}
	if opts != nil {
		if opts.Search != "" {
			variables["search"] = opts.Search
		}
		if opts.Limit > 0 {
			variables["limit"] = opts.Limit
		}
		if opts.Offset > 0 {
			variables["offset"] = opts.Offset
		}
		if opts.OrderBy != "" {
			variables["orderBy"] = opts.OrderBy
		}
		if opts.HasDefaultCategory != nil {
			variables["filters"] = map[string]any{"hasDefaultCategory": *opts.HasDefaultCategory}
		}
		if len(opts.IncludeIDs) > 0 {
			variables["includeIds"] = opts.IncludeIDs
		}
		if opts.IncludeMerchantsWithoutTransactions != nil {
			variables["includeMerchantsWithoutTransactions"] = *opts.IncludeMerchantsWithoutTransactions
		}
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_ListMerchants",
		Query:         ListMerchantsQuery,
		Variables:     variables,
	}, &resp)
	if err != nil {
		return nil, err
	}

	out := make([]*Merchant, 0, len(resp.Merchants))
	for _, r := range resp.Merchants {
		if r != nil {
			out = append(out, toMerchant(r))
		}
	}
	return out, nil
}

func (s *Service) GetMerchant(ctx context.Context, id string) (*Merchant, error) {
	var resp struct {
		Merchant *rawMerchant `json:"merchant"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_GetEditMerchant",
		Query:         GetMerchantQuery,
		Variables:     map[string]any{"merchantId": id},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Merchant == nil {
		return nil, errors.New(errors.ResourceNotFound, "merchant not found", errors.CatAPI, false, nil)
	}
	return toMerchant(resp.Merchant), nil
}

type UpdateMerchantInput struct {
	ID                        string
	Name                      string
	DefaultCategoryID         string
	DefaultCategoryMode       string
	ShowDefaultCategoryPrompt *bool
	Recurrence                map[string]any
}

func (s *Service) UpdateMerchant(ctx context.Context, input *UpdateMerchantInput) (*Merchant, error) {
	var resp struct {
		UpdateMerchant struct {
			Merchant *rawMerchant   `json:"merchant"`
			Errors   []payloadError `json:"errors"`
		} `json:"updateMerchant"`
	}

	merchantInput := map[string]any{"merchantId": input.ID}
	if input.Name != "" {
		merchantInput["name"] = input.Name
	}
	if input.DefaultCategoryID != "" {
		merchantInput["defaultCategoryId"] = input.DefaultCategoryID
	}
	if input.DefaultCategoryMode != "" {
		merchantInput["defaultCategoryApplicationMode"] = input.DefaultCategoryMode
	}
	if input.ShowDefaultCategoryPrompt != nil {
		merchantInput["showDefaultCategoryPrompt"] = *input.ShowDefaultCategoryPrompt
	}
	if input.Recurrence != nil {
		merchantInput["recurrence"] = input.Recurrence
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateMerchant",
		Query:         UpdateMerchantMutation,
		Variables: map[string]any{
			"input": merchantInput,
		},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.UpdateMerchant.Errors, "failed to update merchant"); apiErr != nil {
		return nil, apiErr
	}
	if resp.UpdateMerchant.Merchant == nil {
		return nil, errors.New(errors.APISchemaChanged, "merchant update response missing merchant", errors.CatAPI, false, nil)
	}
	return toMerchant(resp.UpdateMerchant.Merchant), nil
}

func (s *Service) DeleteMerchant(ctx context.Context, id, moveToID string) error {
	variables := map[string]any{"merchantId": id}
	if moveToID != "" {
		variables["moveToId"] = moveToID
	}
	var resp struct {
		DeleteMerchant struct {
			Success bool `json:"success"`
		} `json:"deleteMerchant"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_DeleteMerchant",
		Query:         DeleteMerchantMutation,
		Variables:     variables,
	}, &resp)
	if err != nil {
		return err
	}
	if !resp.DeleteMerchant.Success {
		return errors.New(errors.APIError, "failed to delete merchant", errors.CatAPI, false, nil)
	}
	return nil
}
