package monarch

import (
	"context"
	_ "embed"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

//go:embed queries/tags/list.graphql
var GetTagsQuery string

type Tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

func (s *Service) ListTags(ctx context.Context) ([]Tag, error) {
	var resp struct {
		HouseholdTransactionTags []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"householdTransactionTags"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetTags",
		Query:         GetTagsQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	tags := make([]Tag, len(resp.HouseholdTransactionTags))
	for i, t := range resp.HouseholdTransactionTags {
		tags[i] = Tag{
			ID:    t.ID,
			Name:  t.Name,
			Color: t.Color,
		}
	}

	return tags, nil
}
