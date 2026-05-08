package monarch

import (
	"context"
	_ "embed"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

//go:embed queries/tags/list.graphql
var GetTagsQuery string

//go:embed queries/tags/create.graphql
var CreateTagMutation string

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

func (s *Service) CreateTag(ctx context.Context, name, color string) (*Tag, error) {
	var resp struct {
		CreateHouseholdTransactionTag struct {
			Tag struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Color string `json:"color"`
			} `json:"tag"`
		} `json:"createHouseholdTransactionTag"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "CreateTag",
		Query:         CreateTagMutation,
		Variables: map[string]interface{}{
			"name":  name,
			"color": color,
		},
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &Tag{
		ID:    resp.CreateHouseholdTransactionTag.Tag.ID,
		Name:  resp.CreateHouseholdTransactionTag.Tag.Name,
		Color: resp.CreateHouseholdTransactionTag.Tag.Color,
	}, nil
}
