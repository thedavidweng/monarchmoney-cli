package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetGoalsQuery = queries.Get("goals/list.graphql")

type Goal struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *Service) ListGoals(ctx context.Context) ([]Goal, error) {
	var resp struct {
		Goals []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"goalsV2"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_GoalsV2",
		Query:         GetGoalsQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}

	goals := make([]Goal, len(resp.Goals))
	for i, goal := range resp.Goals {
		goals[i] = Goal{ID: goal.ID, Name: goal.Name}
	}
	return goals, nil
}
