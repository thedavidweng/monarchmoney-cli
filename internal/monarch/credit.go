package monarch

import (
	"context"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
	"github.com/monarchmoney-cli/monarch/queries"
)

var GetCreditHistoryQuery = queries.Get("credit/history.graphql")

type CreditRecord struct {
	Date  string `json:"date"`
	Score int    `json:"score"`
}

func (s *Service) GetCreditHistory(ctx context.Context) ([]CreditRecord, error) {
	var resp struct {
		CreditScoreHistory []struct {
			Date  string `json:"date"`
			Score int    `json:"score"`
		} `json:"creditScoreHistory"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCreditHistory",
		Query:         GetCreditHistoryQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	history := make([]CreditRecord, len(resp.CreditScoreHistory))
	for i, r := range resp.CreditScoreHistory {
		history[i] = CreditRecord{
			Date:  r.Date,
			Score: r.Score,
		}
	}

	return history, nil
}
