package monarch

import (
	"context"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
	"github.com/monarchmoney-cli/monarch/queries"
)

var GetCategoriesQuery = queries.Get("categories/list.graphql")
var GetCategoryGroupsQuery = queries.Get("categories/groups.graphql")
var CreateCategoryMutation = queries.Get("categories/create.graphql")
var DeleteCategoryMutation = queries.Get("categories/delete.graphql")
var DeleteCategoriesMutation = queries.Get("categories/delete_many.graphql")


type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	GroupName string `json:"group_name"`
}

type CategoryGroup struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Categories []Category `json:"categories,omitempty"`
}

func (s *Service) ListCategoryGroups(ctx context.Context) ([]CategoryGroup, error) {
	var resp struct {
		CategoryGroups []struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Type       string `json:"type"`
			Categories []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"categories"`
		} `json:"categoryGroups"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCategoryGroups",
		Query:         GetCategoryGroupsQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	groups := make([]CategoryGroup, len(resp.CategoryGroups))
	for i, g := range resp.CategoryGroups {
		cats := make([]Category, len(g.Categories))
		for j, c := range g.Categories {
			cats[j] = Category{ID: c.ID, Name: c.Name}
		}
		groups[i] = CategoryGroup{
			ID:         g.ID,
			Name:       g.Name,
			Type:       g.Type,
			Categories: cats,
		}
	}

	return groups, nil
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

func (s *Service) CreateCategory(ctx context.Context, name, groupID string) (*Category, error) {
	var resp struct {
		CreateCategory struct {
			Category struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"category"`
		} `json:"createCategory"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "CreateCategory",
		Query:         CreateCategoryMutation,
		Variables: map[string]interface{}{
			"name":    name,
			"groupId": groupID,
		},
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &Category{
		ID:   resp.CreateCategory.Category.ID,
		Name: resp.CreateCategory.Category.Name,
	}, nil
}

func (s *Service) DeleteCategory(ctx context.Context, id string) error {
	var resp struct {
		DeleteCategory struct {
			OK bool `json:"ok"`
		} `json:"deleteCategory"`
	}

	return s.Client.Do(ctx, &graphql.Request{
		OperationName: "DeleteCategory",
		Query:         DeleteCategoryMutation,
		Variables:     map[string]interface{}{"id": id},
	}, &resp)
}

func (s *Service) DeleteCategories(ctx context.Context, ids []string) error {
	var resp struct {
		DeleteTransactionCategories struct {
			OK bool `json:"ok"`
		} `json:"deleteTransactionCategories"`
	}

	return s.Client.Do(ctx, &graphql.Request{
		OperationName: "DeleteCategories",
		Query:         DeleteCategoriesMutation,
		Variables:     map[string]interface{}{"ids": ids},
	}, &resp)
}
