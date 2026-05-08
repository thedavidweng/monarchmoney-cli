package monarch

import (
	"context"
	_ "embed"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

//go:embed queries/categories/list.graphql
var GetCategoriesQuery string

type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	GroupName string `json:"group_name"`
}

func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	var resp struct {
		Categories []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Group struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"group"`
		} `json:"categories"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCategories",
		Query:         GetCategoriesQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	cats := make([]Category, len(resp.Categories))
	for i, c := range resp.Categories {
		cats[i] = Category{
			ID:        c.ID,
			Name:      c.Name,
			GroupName: c.Group.Name,
		}
	}

	return cats, nil
}
