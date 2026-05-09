package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetCreditHistoryQuery = queries.Get("credit/history.graphql")

type CreditRecord struct {
	Date   string `json:"date"`
	Score  int    `json:"score"`
	UserID string `json:"user_id"`
}

func (s *Service) GetCreditHistory(ctx context.Context) ([]CreditRecord, error) {
	var resp struct {
		CreditScoreSnapshots []struct {
			ReportedDate string `json:"reportedDate"`
			Score        int    `json:"score"`
			User         struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"creditScoreSnapshots"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCreditScoreSnapshots",
		Query:         GetCreditHistoryQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	history := make([]CreditRecord, len(resp.CreditScoreSnapshots))
	for i, r := range resp.CreditScoreSnapshots {
		history[i] = CreditRecord{
			Date:   r.ReportedDate,
			Score:  r.Score,
			UserID: r.User.ID,
		}
	}

	return history, nil
}
