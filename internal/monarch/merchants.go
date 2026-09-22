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

type Merchant struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	LogoURL           string `json:"logo_url,omitempty"`
	TransactionCount  int    `json:"transaction_count"`
	RuleCount         int    `json:"rule_count,omitempty"`
	CanBeDeleted      *bool  `json:"can_be_deleted,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
	RecurringStreamID string `json:"recurring_stream_id,omitempty"`
}

type rawMerchant struct {
	ID                         string `json:"id"`
	Name                       string `json:"name"`
	LogoURL                    string `json:"logoUrl"`
	TransactionCount           int    `json:"transactionCount"`
	RuleCount                  int    `json:"ruleCount"`
	CanBeDeleted               *bool  `json:"canBeDeleted"`
	CreatedAt                  string `json:"createdAt"`
	RecurringTransactionStream *struct {
		ID string `json:"id"`
	} `json:"recurringTransactionStream"`
}

func toMerchant(r *rawMerchant) *Merchant {
	m := &Merchant{
		ID:               r.ID,
		Name:             r.Name,
		LogoURL:          r.LogoURL,
		TransactionCount: r.TransactionCount,
		RuleCount:        r.RuleCount,
		CanBeDeleted:     r.CanBeDeleted,
		CreatedAt:        r.CreatedAt,
	}
	if r.RecurringTransactionStream != nil {
		m.RecurringStreamID = r.RecurringTransactionStream.ID
	}
	return m
}

func (s *Service) ListMerchants(ctx context.Context, search string, limit, offset int, orderBy string) ([]*Merchant, error) {
	var resp struct {
		Merchants []*rawMerchant `json:"merchants"`
	}

	variables := map[string]any{}
	if search != "" {
		variables["search"] = search
	}
	if limit > 0 {
		variables["limit"] = limit
	}
	if offset > 0 {
		variables["offset"] = offset
	}
	if orderBy != "" {
		variables["orderBy"] = orderBy
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

func (s *Service) UpdateMerchant(ctx context.Context, id, name string) (*Merchant, error) {
	var resp struct {
		UpdateMerchant struct {
			Merchant *rawMerchant   `json:"merchant"`
			Errors   []payloadError `json:"errors"`
		} `json:"updateMerchant"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateMerchant",
		Query:         UpdateMerchantMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"merchantId": id,
				"name":       name,
			},
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
