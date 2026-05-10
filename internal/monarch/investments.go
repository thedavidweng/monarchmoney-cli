package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetInvestmentPortfolioQuery = queries.Get("investments/portfolio.graphql")
var GetSecurityPerformanceQuery = queries.Get("investments/performance.graphql")

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

	input := map[string]interface{}{}
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
		Variables:     map[string]interface{}{"portfolioInput": input},
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
		for _, holding := range node.Holdings {
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
		Variables: map[string]interface{}{
			"input": map[string]interface{}{
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
