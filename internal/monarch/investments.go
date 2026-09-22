package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var (
	GetInvestmentPortfolioQuery = queries.Get("investments/portfolio.graphql")
	GetSecurityPerformanceQuery = queries.Get("investments/performance.graphql")
	ListInvestmentAccountsQuery = queries.Get("investments/accounts.graphql")
	ListHoldingsQuery           = queries.Get("accounts/holdings.graphql")
	SearchSecuritiesQuery       = queries.Get("investments/securities.graphql")
	GetSecurityQuery            = queries.Get("investments/security.graphql")
	CreateManualHoldingMutation = queries.Get("investments/holdings_create.graphql")
	UpdateHoldingMutation       = queries.Get("investments/holdings_update.graphql")
	DeleteHoldingMutation       = queries.Get("investments/holdings_delete.graphql")
)

type InvestmentPortfolioOptions struct {
	StartDate  string
	EndDate    string
	AccountIDs []string
}

type InvestmentPortfolio struct {
	Performance InvestmentPerformance   `json:"performance"`
	Holdings    []InvestmentHoldingNode `json:"holdings"`
}

type InvestmentPerformance struct {
	TotalValue         float64 `json:"total_value"`
	TotalChangePercent float64 `json:"total_change_percent"`
	TotalChangeDollars float64 `json:"total_change_dollars"`
}

type InvestmentHoldingNode struct {
	ID         string              `json:"id"`
	Quantity   float64             `json:"quantity"`
	Basis      float64             `json:"basis"`
	TotalValue float64             `json:"total_value"`
	Security   InvestmentSecurity  `json:"security"`
	Holdings   []InvestmentHolding `json:"holdings"`
}

type InvestmentSecurity struct {
	ID           string  `json:"id"`
	Ticker       string  `json:"ticker"`
	Name         string  `json:"name"`
	CurrentPrice float64 `json:"current_price,omitempty"`
}

type InvestmentHolding struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	TypeDisplay string            `json:"type_display"`
	Name        string            `json:"name"`
	Ticker      string            `json:"ticker"`
	Quantity    float64           `json:"quantity"`
	Value       float64           `json:"value"`
	Account     InvestmentAccount `json:"account"`
}

type InvestmentAccount struct {
	ID                    string `json:"id"`
	DisplayName           string `json:"display_name"`
	AccountType           string `json:"account_type"`
	AccountTypeDisplay    string `json:"account_type_display"`
	AccountSubtype        string `json:"account_subtype"`
	AccountSubtypeDisplay string `json:"account_subtype_display"`
}

type SecurityPerformanceOptions struct {
	SecurityIDs   []string
	StartDate     string
	EndDate       string
	IncludeValues bool
}

type SecurityPerformance struct {
	Security        InvestmentSecurity         `json:"security"`
	HistoricalChart []SecurityPerformancePoint `json:"historical_chart"`
}

type SecurityPerformancePoint struct {
	Date          string   `json:"date"`
	ReturnPercent float64  `json:"return_percent"`
	Value         *float64 `json:"value,omitempty"`
}

func (s *Service) GetInvestmentPortfolio(ctx context.Context, opts InvestmentPortfolioOptions) (*InvestmentPortfolio, error) {
	var resp struct {
		Portfolio struct {
			Performance struct {
				TotalValue         float64 `json:"totalValue"`
				TotalChangePercent float64 `json:"totalChangePercent"`
				TotalChangeDollars float64 `json:"totalChangeDollars"`
			} `json:"performance"`
			AggregateHoldings struct {
				Edges []struct {
					Node struct {
						ID         string  `json:"id"`
						Quantity   float64 `json:"quantity"`
						Basis      float64 `json:"basis"`
						TotalValue float64 `json:"totalValue"`
						Security   struct {
							ID           string  `json:"id"`
							Ticker       string  `json:"ticker"`
							Name         string  `json:"name"`
							CurrentPrice float64 `json:"currentPrice"`
						} `json:"security"`
						Holdings []struct {
							ID          string  `json:"id"`
							Type        string  `json:"type"`
							TypeDisplay string  `json:"typeDisplay"`
							Name        string  `json:"name"`
							Ticker      string  `json:"ticker"`
							Quantity    float64 `json:"quantity"`
							Value       float64 `json:"value"`
							Account     struct {
								ID          string `json:"id"`
								DisplayName string `json:"displayName"`
								Type        struct {
									Name    string `json:"name"`
									Display string `json:"display"`
								} `json:"type"`
								Subtype struct {
									Name    string `json:"name"`
									Display string `json:"display"`
								} `json:"subtype"`
							} `json:"account"`
						} `json:"holdings"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"aggregateHoldings"`
		} `json:"portfolio"`
	}

	input := map[string]any{}
	if opts.StartDate != "" {
		input["startDate"] = opts.StartDate
	}
	if opts.EndDate != "" {
		input["endDate"] = opts.EndDate
	}
	if len(opts.AccountIDs) > 0 {
		input["accounts"] = opts.AccountIDs
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_GetPortfolio",
		Query:         GetInvestmentPortfolioQuery,
		Variables:     map[string]any{"portfolioInput": input},
	}, &resp)
	if err != nil {
		return nil, err
	}

	portfolio := &InvestmentPortfolio{
		Performance: InvestmentPerformance{
			TotalValue:         resp.Portfolio.Performance.TotalValue,
			TotalChangePercent: resp.Portfolio.Performance.TotalChangePercent,
			TotalChangeDollars: resp.Portfolio.Performance.TotalChangeDollars,
		},
		Holdings: make([]InvestmentHoldingNode, 0, len(resp.Portfolio.AggregateHoldings.Edges)),
	}

	for _, edge := range resp.Portfolio.AggregateHoldings.Edges {
		node := edge.Node
		holdingNode := InvestmentHoldingNode{
			ID:         node.ID,
			Quantity:   node.Quantity,
			Basis:      node.Basis,
			TotalValue: node.TotalValue,
			Security: InvestmentSecurity{
				ID:           node.Security.ID,
				Ticker:       node.Security.Ticker,
				Name:         node.Security.Name,
				CurrentPrice: node.Security.CurrentPrice,
			},
			Holdings: make([]InvestmentHolding, 0, len(node.Holdings)),
		}
		for j := range node.Holdings {
			holding := &node.Holdings[j]
			holdingNode.Holdings = append(holdingNode.Holdings, InvestmentHolding{
				ID:          holding.ID,
				Type:        holding.Type,
				TypeDisplay: holding.TypeDisplay,
				Name:        holding.Name,
				Ticker:      holding.Ticker,
				Quantity:    holding.Quantity,
				Value:       holding.Value,
				Account: InvestmentAccount{
					ID:                    holding.Account.ID,
					DisplayName:           holding.Account.DisplayName,
					AccountType:           holding.Account.Type.Name,
					AccountTypeDisplay:    holding.Account.Type.Display,
					AccountSubtype:        holding.Account.Subtype.Name,
					AccountSubtypeDisplay: holding.Account.Subtype.Display,
				},
			})
		}
		portfolio.Holdings = append(portfolio.Holdings, holdingNode)
	}

	return portfolio, nil
}

func (s *Service) GetSecurityPerformance(ctx context.Context, opts SecurityPerformanceOptions) ([]SecurityPerformance, error) {
	var resp struct {
		SecurityHistoricalPerformance []struct {
			Security struct {
				ID     string `json:"id"`
				Ticker string `json:"ticker"`
				Name   string `json:"name"`
			} `json:"security"`
			HistoricalChart []struct {
				Date          string   `json:"date"`
				ReturnPercent float64  `json:"returnPercent"`
				Value         *float64 `json:"value"`
			} `json:"historicalChart"`
		} `json:"securityHistoricalPerformance"`
	}

	operationName := "Web_GetSecuritiesHistoricalPerformance"
	query := securitiesHistoricalPerformanceQuery(false)
	if opts.IncludeValues {
		operationName = "Web_GetInvestmentsHoldingDrawerHistoricalPerformance"
		query = GetSecurityPerformanceQuery
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: operationName,
		Query:         query,
		Variables: map[string]any{
			"input": map[string]any{
				"securityIds": opts.SecurityIDs,
				"startDate":   opts.StartDate,
				"endDate":     opts.EndDate,
			},
		},
	}, &resp)
	if err != nil {
		return nil, err
	}

	out := make([]SecurityPerformance, 0, len(resp.SecurityHistoricalPerformance))
	for _, item := range resp.SecurityHistoricalPerformance {
		performance := SecurityPerformance{
			Security: InvestmentSecurity{
				ID:     item.Security.ID,
				Ticker: item.Security.Ticker,
				Name:   item.Security.Name,
			},
			HistoricalChart: make([]SecurityPerformancePoint, 0, len(item.HistoricalChart)),
		}
		for _, point := range item.HistoricalChart {
			outPoint := SecurityPerformancePoint{
				Date:          point.Date,
				ReturnPercent: point.ReturnPercent,
			}
			if opts.IncludeValues {
				outPoint.Value = point.Value
			}
			performance.HistoricalChart = append(performance.HistoricalChart, outPoint)
		}
		out = append(out, performance)
	}

	return out, nil
}

func securitiesHistoricalPerformanceQuery(includeValues bool) string {
	valueField := ""
	if includeValues {
		valueField = "\n      value"
	}
	return `query Web_GetSecuritiesHistoricalPerformance($input: SecurityHistoricalPerformanceInput!) {
  securityHistoricalPerformance(input: $input) {
    security {
      id
      ticker
      name
    }
    historicalChart {
      date
      returnPercent` + valueField + `
    }
  }
}`
}

type Security struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Ticker       string  `json:"ticker"`
	Type         string  `json:"type,omitempty"`
	TypeDisplay  string  `json:"type_display,omitempty"`
	CurrentPrice float64 `json:"current_price,omitempty"`
}

type HoldingDetail struct {
	ID          string  `json:"id"`
	SecurityID  string  `json:"security_id,omitempty"`
	Ticker      string  `json:"ticker,omitempty"`
	Name        string  `json:"name,omitempty"`
	Quantity    float64 `json:"quantity"`
	TotalValue  float64 `json:"total_value"`
	AccountID   string  `json:"account_id,omitempty"`
	AccountName string  `json:"account_name,omitempty"`
}

type rawSecurity struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Ticker       string  `json:"ticker"`
	Type         string  `json:"type"`
	TypeDisplay  string  `json:"typeDisplay"`
	CurrentPrice float64 `json:"currentPrice"`
}

func toSecurity(r *rawSecurity) *Security {
	return &Security{
		ID: r.ID, Name: r.Name, Ticker: r.Ticker,
		Type: r.Type, TypeDisplay: r.TypeDisplay, CurrentPrice: r.CurrentPrice,
	}
}

func (s *Service) ListInvestmentAccounts(ctx context.Context) ([]InvestmentAccount, error) {
	var resp struct {
		Accounts []struct {
			ID                string `json:"id"`
			DisplayName       string `json:"displayName"`
			IncludeInNetWorth bool   `json:"includeInNetWorth"`
		} `json:"accounts"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_GetInvestmentsAccounts",
		Query:         ListInvestmentAccountsQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}

	out := make([]InvestmentAccount, 0, len(resp.Accounts))
	for _, a := range resp.Accounts {
		out = append(out, InvestmentAccount{ID: a.ID, DisplayName: a.DisplayName})
	}
	return out, nil
}

func (s *Service) ListHoldingDetails(ctx context.Context, accountIDs []string) ([]*HoldingDetail, error) {
	var resp struct {
		Portfolio *struct {
			AggregateHoldings *struct {
				Edges []struct {
					Node *struct {
						ID         string  `json:"id"`
						Quantity   float64 `json:"quantity"`
						TotalValue float64 `json:"totalValue"`
						Security   *struct {
							ID           string  `json:"id"`
							Name         string  `json:"name"`
							Ticker       string  `json:"ticker"`
							CurrentPrice float64 `json:"currentPrice"`
						} `json:"security"`
						Holdings []struct {
							ID       string  `json:"id"`
							Name     string  `json:"name"`
							Ticker   string  `json:"ticker"`
							Quantity float64 `json:"quantity"`
							Value    float64 `json:"value"`
							Account  *struct {
								ID          string `json:"id"`
								DisplayName string `json:"displayName"`
							} `json:"account"`
						} `json:"holdings"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"aggregateHoldings"`
		} `json:"portfolio"`
	}

	input := map[string]any{}
	if len(accountIDs) > 0 {
		input["accountIds"] = accountIDs
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_GetHoldings",
		Query:         ListHoldingsQuery,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Portfolio == nil || resp.Portfolio.AggregateHoldings == nil {
		return []*HoldingDetail{}, nil
	}

	out := make([]*HoldingDetail, 0)
	for _, edge := range resp.Portfolio.AggregateHoldings.Edges {
		node := edge.Node
		if node == nil {
			continue
		}
		for _, h := range node.Holdings {
			holding := &HoldingDetail{
				ID: h.ID, Ticker: h.Ticker, Name: h.Name,
				Quantity: h.Quantity, TotalValue: h.Value,
			}
			if h.Account != nil {
				holding.AccountID, holding.AccountName = h.Account.ID, h.Account.DisplayName
			}
			if node.Security != nil {
				holding.SecurityID = node.Security.ID
			}
			out = append(out, holding)
		}
	}
	return out, nil
}

func (s *Service) GetHoldingDetail(ctx context.Context, id string) (*HoldingDetail, error) {
	holdings, err := s.ListHoldingDetails(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, h := range holdings {
		if h.ID == id {
			return h, nil
		}
	}
	return nil, errors.New(errors.ResourceNotFound, "holding not found", errors.CatAPI, false, nil)
}

func (s *Service) SearchSecurities(ctx context.Context, query string, limit int) ([]*Security, error) {
	if limit <= 0 {
		limit = 20
	}
	var resp struct {
		Securities []*rawSecurity `json:"securities"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_SearchSecurities",
		Query:         SearchSecuritiesQuery,
		Variables: map[string]any{
			"search":            query,
			"limit":             limit,
			"orderByPopularity": true,
		},
	}, &resp)
	if err != nil {
		return nil, err
	}

	out := make([]*Security, 0, len(resp.Securities))
	for _, r := range resp.Securities {
		if r != nil {
			out = append(out, toSecurity(r))
		}
	}
	return out, nil
}

func (s *Service) GetSecurity(ctx context.Context, id string) (*Security, error) {
	var resp struct {
		Security *rawSecurity `json:"security"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetHoldingDetailsFormSecurityDetails",
		Query:         GetSecurityQuery,
		Variables:     map[string]any{"id": id},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Security == nil {
		return nil, errors.New(errors.ResourceNotFound, "security not found", errors.CatAPI, false, nil)
	}
	return toSecurity(resp.Security), nil
}

type ManualHoldingInput struct {
	AccountID  string
	SecurityID string
	Quantity   float64
	CostBasis  *float64
}

func (s *Service) CreateManualHolding(ctx context.Context, input *ManualHoldingInput) (*HoldingDetail, error) {
	var resp struct {
		CreateManualHolding struct {
			Holding *struct {
				ID     string `json:"id"`
				Ticker string `json:"ticker"`
			} `json:"holding"`
			Errors []payloadError `json:"errors"`
		} `json:"createManualHolding"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_CreateManualHolding",
		Query:         CreateManualHoldingMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"accountId":  input.AccountID,
				"securityId": input.SecurityID,
				"quantity":   input.Quantity,
			},
		},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.CreateManualHolding.Errors, "failed to create manual holding"); apiErr != nil {
		return nil, apiErr
	}
	if resp.CreateManualHolding.Holding == nil || resp.CreateManualHolding.Holding.ID == "" {
		return nil, errors.New(errors.APISchemaChanged, "manual holding creation response missing holding", errors.CatAPI, false, nil)
	}
	holdingID := resp.CreateManualHolding.Holding.ID
	if input.CostBasis != nil {
		if _, err := s.UpdateManualHolding(ctx, holdingID, &ManualHoldingUpdate{CostBasis: input.CostBasis}); err != nil {
			return nil, err
		}
	}
	holding, err := s.GetHoldingDetail(ctx, holdingID)
	if err != nil {
		return nil, err
	}
	return holding, nil
}

type ManualHoldingUpdate struct {
	Quantity     *float64
	CostBasis    *float64
	SecurityType *string
}

func (s *Service) UpdateManualHolding(ctx context.Context, id string, update *ManualHoldingUpdate) (*HoldingDetail, error) {
	input := map[string]any{"id": id}
	if update.Quantity != nil {
		input["quantity"] = *update.Quantity
	}
	if update.CostBasis != nil {
		input["userCostBasis"] = *update.CostBasis
	}
	if update.SecurityType != nil {
		input["securityType"] = *update.SecurityType
	}

	var resp struct {
		UpdateHolding struct {
			Errors []payloadError `json:"errors"`
		} `json:"updateHolding"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateHolding",
		Query:         UpdateHoldingMutation,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.UpdateHolding.Errors, "failed to update holding"); apiErr != nil {
		return nil, apiErr
	}
	holding, err := s.GetHoldingDetail(ctx, id)
	if err != nil {
		return nil, err
	}
	return holding, nil
}

func (s *Service) DeleteManualHolding(ctx context.Context, id string) error {
	var resp struct {
		DeleteHolding struct {
			Deleted bool           `json:"deleted"`
			Errors  []payloadError `json:"errors"`
		} `json:"deleteHolding"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_DeleteHolding",
		Query:         DeleteHoldingMutation,
		Variables:     map[string]any{"id": id},
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.DeleteHolding.Errors, "failed to delete holding"); apiErr != nil {
		return apiErr
	}
	if !resp.DeleteHolding.Deleted {
		return errors.New(errors.APIError, "failed to delete holding", errors.CatAPI, false, nil)
	}
	return nil
}
