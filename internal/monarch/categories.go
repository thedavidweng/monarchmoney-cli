package monarch

import (
	"context"
	"fmt"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var (
	GetCategoriesQuery           = queries.Get("categories/list.graphql")
	GetCategoryGroupsQuery       = queries.Get("categories/groups.graphql")
	CreateCategoryMutation       = queries.Get("categories/create.graphql")
	DeleteCategoryMutation       = queries.Get("categories/delete.graphql")
	DeleteCategoriesMutation     = queries.Get("categories/delete_many.graphql")
	UpdateCategoryMutation       = queries.Get("categories/update.graphql")
	GetCategoryRolloverQuery     = queries.Get("categories/rollover.graphql")
	UpdateCategoryGroupMutation  = queries.Get("categories/update_group.graphql")
	GetCategoryQuery             = queries.Get("categories/show.graphql")
	ReactivateCategoryMutation   = queries.Get("categories/reactivate.graphql")
	ReorderCategoryMutation      = queries.Get("categories/reorder.graphql")
	CreateCategoryGroupMutation  = queries.Get("categories/create_group.graphql")
	DeleteCategoryGroupMutation  = queries.Get("categories/delete_group.graphql")
	ReorderCategoryGroupMutation = queries.Get("categories/reorder_group.graphql")
)

type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	GroupName string `json:"group_name"`
	GroupID   string `json:"group_id"`
	GroupType string `json:"group_type"`
	Order     int    `json:"order"`
	Icon      string `json:"icon"`
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
		OperationName: "ManageGetCategoryGroups",
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
			Order int    `json:"order"`
			Icon  string `json:"icon"`
			Group struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
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
			GroupID:   c.Group.ID,
			GroupType: c.Group.Type,
			Order:     c.Order,
			Icon:      c.Icon,
		}
	}

	return cats, nil
}

func (s *Service) CreateCategory(ctx context.Context, name, groupID, icon string) (*Category, error) {
	var resp struct {
		CreateCategory struct {
			Category *rawCategory   `json:"category"`
			Errors   []payloadError `json:"errors"`
		} `json:"createCategory"`
	}

	input := map[string]any{"name": name, "group": groupID}
	if icon != "" {
		input["icon"] = icon
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_CreateCategory",
		Query:         CreateCategoryMutation,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.CreateCategory.Errors, "failed to create category"); apiErr != nil {
		return nil, apiErr
	}
	if resp.CreateCategory.Category == nil {
		return nil, errors.New(errors.APISchemaChanged, "category creation response missing category", errors.CatAPI, false, nil)
	}
	return toCategory(resp.CreateCategory.Category), nil
}

func (s *Service) DeleteCategory(ctx context.Context, id, moveToID string) error {
	variables := map[string]any{"id": id}
	if moveToID != "" {
		variables["moveToCategoryId"] = moveToID
	}
	var resp struct {
		DeleteCategory struct {
			Deleted bool           `json:"deleted"`
			Errors  []payloadError `json:"errors"`
		} `json:"deleteCategory"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_DeleteCategory",
		Query:         DeleteCategoryMutation,
		Variables:     variables,
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.DeleteCategory.Errors, "failed to delete category"); apiErr != nil {
		return apiErr
	}
	if !resp.DeleteCategory.Deleted {
		return errors.New(errors.APIError, "failed to delete category", errors.CatAPI, false, nil)
	}
	return nil
}

func (s *Service) DeleteCategories(ctx context.Context, ids []string) error {
	var resp struct {
		DeleteTransactionCategories struct {
			OK bool `json:"ok"`
		} `json:"deleteTransactionCategories"`
	}

	return s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "DeleteCategories",
		Query:         DeleteCategoriesMutation,
		Variables:     map[string]any{"ids": ids},
	}, &resp)
}

type CategoryRollover struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	StartMonth      string  `json:"start_month"`
	StartingBalance float64 `json:"starting_balance"`
	Type            string  `json:"type"`
	Frequency       string  `json:"frequency"`
	TargetAmount    float64 `json:"target_amount"`
}

type UpdateCategoryOptions struct {
	Name              *string
	Icon              *string
	BudgetVariability *string
	ExcludeFromBudget *bool
}

func (s *Service) UpdateCategory(ctx context.Context, categoryID string, opts UpdateCategoryOptions) (*Category, error) {
	input := map[string]any{"id": categoryID}
	if opts.Name != nil {
		input["name"] = *opts.Name
	}
	if opts.Icon != nil {
		input["icon"] = *opts.Icon
	}
	if opts.BudgetVariability != nil {
		input["budgetVariability"] = *opts.BudgetVariability
	}
	if opts.ExcludeFromBudget != nil {
		input["excludeFromBudget"] = *opts.ExcludeFromBudget
	}

	var resp struct {
		UpdateCategory struct {
			Errors []struct {
				Message string `json:"message"`
				Code    string `json:"code"`
			} `json:"errors"`
			Category struct {
				ID                string `json:"id"`
				Name              string `json:"name"`
				Icon              string `json:"icon"`
				BudgetVariability string `json:"budgetVariability"`
				ExcludeFromBudget bool   `json:"excludeFromBudget"`
				Group             struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"group"`
			} `json:"category"`
		} `json:"updateCategory"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_UpdateCategory",
		Query:         UpdateCategoryMutation,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.UpdateCategory.Errors) > 0 {
		return nil, fmt.Errorf("update category failed: %s", resp.UpdateCategory.Errors[0].Message)
	}

	return &Category{
		ID:        resp.UpdateCategory.Category.ID,
		Name:      resp.UpdateCategory.Category.Name,
		Icon:      resp.UpdateCategory.Category.Icon,
		GroupID:   resp.UpdateCategory.Category.Group.ID,
		GroupType: resp.UpdateCategory.Category.Group.Type,
	}, nil
}

func (s *Service) GetCategoryRollover(ctx context.Context, categoryID string) (*CategoryRollover, error) {
	var resp struct {
		Category struct {
			ID             string `json:"id"`
			Name           string `json:"name"`
			RolloverPeriod *struct {
				ID              string  `json:"id"`
				StartMonth      string  `json:"startMonth"`
				StartingBalance float64 `json:"startingBalance"`
				Type            string  `json:"type"`
				Frequency       string  `json:"frequency"`
				TargetAmount    float64 `json:"targetAmount"`
			} `json:"rolloverPeriod"`
		} `json:"category"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCategoryRollover",
		Query:         GetCategoryRolloverQuery,
		Variables:     map[string]any{"id": categoryID},
	}, &resp)
	if err != nil {
		return nil, err
	}

	r := &CategoryRollover{ID: resp.Category.ID, Name: resp.Category.Name}
	if resp.Category.RolloverPeriod != nil {
		r.StartMonth = resp.Category.RolloverPeriod.StartMonth
		r.StartingBalance = resp.Category.RolloverPeriod.StartingBalance
		r.Type = resp.Category.RolloverPeriod.Type
		r.Frequency = resp.Category.RolloverPeriod.Frequency
		r.TargetAmount = resp.Category.RolloverPeriod.TargetAmount
	}
	return r, nil
}

type UpdateCategoryGroupOptions struct {
	Name                       *string
	BudgetVariability          *string
	GroupLevelBudgetingEnabled *bool
	RolloverEnabled            *bool
	RolloverStartMonth         *string
	RolloverStartingBalance    *float64
	RolloverType               *string
}

func (s *Service) UpdateCategoryGroup(ctx context.Context, groupID string, opts UpdateCategoryGroupOptions) (*CategoryGroup, error) {
	input := map[string]any{"id": groupID}
	if opts.Name != nil {
		input["name"] = *opts.Name
	}
	if opts.BudgetVariability != nil {
		input["budgetVariability"] = *opts.BudgetVariability
	}
	if opts.GroupLevelBudgetingEnabled != nil {
		input["groupLevelBudgetingEnabled"] = *opts.GroupLevelBudgetingEnabled
	}
	if opts.RolloverEnabled != nil {
		input["rolloverEnabled"] = *opts.RolloverEnabled
	}
	if opts.RolloverStartMonth != nil {
		input["rolloverStartMonth"] = *opts.RolloverStartMonth
	}
	if opts.RolloverStartingBalance != nil {
		input["rolloverStartingBalance"] = *opts.RolloverStartingBalance
	}
	if opts.RolloverType != nil {
		input["rolloverType"] = *opts.RolloverType
	}

	var resp struct {
		UpdateCategoryGroup struct {
			CategoryGroup struct {
				ID                         string `json:"id"`
				Name                       string `json:"name"`
				Order                      int    `json:"order"`
				Type                       string `json:"type"`
				Color                      string `json:"color"`
				GroupLevelBudgetingEnabled bool   `json:"groupLevelBudgetingEnabled"`
				BudgetVariability          string `json:"budgetVariability"`
			} `json:"categoryGroup"`
		} `json:"updateCategoryGroup"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateCategoryGroup",
		Query:         UpdateCategoryGroupMutation,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return nil, err
	}

	return &CategoryGroup{
		ID:   resp.UpdateCategoryGroup.CategoryGroup.ID,
		Name: resp.UpdateCategoryGroup.CategoryGroup.Name,
		Type: resp.UpdateCategoryGroup.CategoryGroup.Type,
	}, nil
}

type rawCategory struct {
	ID    string `json:"id"`
	Order int    `json:"order"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Group *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"group"`
}

func toCategory(r *rawCategory) *Category {
	c := &Category{ID: r.ID, Order: r.Order, Name: r.Name, Icon: r.Icon}
	if r.Group != nil {
		c.GroupID = r.Group.ID
		c.GroupName = r.Group.Name
		c.GroupType = r.Group.Type
	}
	return c
}

func (s *Service) GetCategory(ctx context.Context, id string) (*Category, error) {
	var resp struct {
		Category *rawCategory `json:"category"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_GetEditCategory",
		Query:         GetCategoryQuery,
		Variables:     map[string]any{"id": id},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Category == nil {
		return nil, errors.New(errors.ResourceNotFound, "category not found", errors.CatAPI, false, nil)
	}
	return toCategory(resp.Category), nil
}

func (s *Service) ReactivateCategory(ctx context.Context, id string) (*Category, error) {
	var resp struct {
		RestoreCategory struct {
			Category *rawCategory   `json:"category"`
			Errors   []payloadError `json:"errors"`
		} `json:"restoreCategory"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_RestoreCategory",
		Query:         ReactivateCategoryMutation,
		Variables:     map[string]any{"id": id},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.RestoreCategory.Errors, "failed to reactivate category"); apiErr != nil {
		return nil, apiErr
	}
	if resp.RestoreCategory.Category == nil {
		return nil, errors.New(errors.APISchemaChanged, "category reactivation response missing category", errors.CatAPI, false, nil)
	}
	return toCategory(resp.RestoreCategory.Category), nil
}

func (s *Service) ReorderCategory(ctx context.Context, id, groupID string, order int) (*Category, error) {
	var resp struct {
		UpdateCategoryOrderInCategoryGroup struct {
			Category *rawCategory `json:"category"`
		} `json:"updateCategoryOrderInCategoryGroup"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_UpdateCategoryOrder",
		Query:         ReorderCategoryMutation,
		Variables: map[string]any{
			"id":              id,
			"categoryGroupId": groupID,
			"order":           order,
		},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.UpdateCategoryOrderInCategoryGroup.Category == nil {
		return nil, errors.New(errors.APISchemaChanged, "category reorder response missing category", errors.CatAPI, false, nil)
	}
	return toCategory(resp.UpdateCategoryOrderInCategoryGroup.Category), nil
}

func (s *Service) CreateCategoryGroup(ctx context.Context, name, groupType string) (*CategoryGroup, error) {
	var resp struct {
		CreateCategoryGroup struct {
			CategoryGroup *struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Order int    `json:"order"`
				Type  string `json:"type"`
			} `json:"categoryGroup"`
		} `json:"createCategoryGroup"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_CreateCategoryGroup",
		Query:         CreateCategoryGroupMutation,
		Variables: map[string]any{
			"input": map[string]any{"name": name, "type": groupType},
		},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.CreateCategoryGroup.CategoryGroup == nil {
		return nil, errors.New(errors.APISchemaChanged, "category group creation response missing categoryGroup", errors.CatAPI, false, nil)
	}
	g := resp.CreateCategoryGroup.CategoryGroup
	return &CategoryGroup{ID: g.ID, Name: g.Name, Type: g.Type}, nil
}

func (s *Service) DeleteCategoryGroup(ctx context.Context, id, moveToID string) error {
	variables := map[string]any{"id": id}
	if moveToID != "" {
		variables["moveToGroupId"] = moveToID
	}
	var resp struct {
		DeleteCategoryGroup struct {
			Deleted bool           `json:"deleted"`
			Errors  []payloadError `json:"errors"`
		} `json:"deleteCategoryGroup"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_DeleteCategoryGroup",
		Query:         DeleteCategoryGroupMutation,
		Variables:     variables,
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.DeleteCategoryGroup.Errors, "failed to delete category group"); apiErr != nil {
		return apiErr
	}
	if !resp.DeleteCategoryGroup.Deleted {
		return errors.New(errors.APIError, "failed to delete category group", errors.CatAPI, false, nil)
	}
	return nil
}

func (s *Service) ReorderCategoryGroup(ctx context.Context, id string, order int) ([]CategoryGroup, error) {
	var resp struct {
		UpdateCategoryGroupOrder struct {
			CategoryGroups []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Order int    `json:"order"`
				Type  string `json:"type"`
			} `json:"categoryGroups"`
		} `json:"updateCategoryGroupOrder"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_UpdateCategoryGroupOrder",
		Query:         ReorderCategoryGroupMutation,
		Variables:     map[string]any{"id": id, "order": order},
	}, &resp)
	if err != nil {
		return nil, err
	}
	out := make([]CategoryGroup, 0, len(resp.UpdateCategoryGroupOrder.CategoryGroups))
	for _, g := range resp.UpdateCategoryGroupOrder.CategoryGroups {
		out = append(out, CategoryGroup{ID: g.ID, Name: g.Name, Type: g.Type})
	}
	return out, nil
}
