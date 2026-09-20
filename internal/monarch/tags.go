package monarch

import (
	"context"
	"sort"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetTagsQuery = queries.Get("tags/list.graphql")
var CreateTagMutation = queries.Get("tags/create.graphql")
var UpdateTagMutation = queries.Get("tags/update.graphql")
var DeleteTagMutation = queries.Get("tags/delete.graphql")
var ReorderTagMutation = queries.Get("tags/reorder.graphql")

type Tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Order int    `json:"order"`
}

type rawTag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Order *int   `json:"order"`
}

type payloadError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func toTag(t *rawTag) Tag {
	tag := Tag{ID: t.ID, Name: t.Name, Color: t.Color, Order: 1 << 30}
	if t.Order != nil {
		tag.Order = *t.Order
	}
	return tag
}

func sortTags(tags []Tag) {
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Order != tags[j].Order {
			return tags[i].Order < tags[j].Order
		}
		if tags[i].Name != tags[j].Name {
			return tags[i].Name < tags[j].Name
		}
		return tags[i].ID < tags[j].ID
	})
}

func (s *Service) ListTags(ctx context.Context, search string, limit int) ([]Tag, error) {
	var resp struct {
		HouseholdTransactionTags []*rawTag `json:"householdTransactionTags"`
	}

	variables := map[string]any{}
	if search != "" {
		variables["search"] = search
	}
	if limit > 0 {
		variables["limit"] = limit
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_GetHouseholdTransactionTags",
		Query:         GetTagsQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	tags := make([]Tag, 0, len(resp.HouseholdTransactionTags))
	for _, t := range resp.HouseholdTransactionTags {
		if t != nil {
			tags = append(tags, toTag(t))
		}
	}
	sortTags(tags)

	return tags, nil
}

func (s *Service) GetTag(ctx context.Context, id string) (*Tag, error) {
	tags, err := s.ListTags(ctx, "", 0)
	if err != nil {
		return nil, err
	}
	for i := range tags {
		if tags[i].ID == id {
			return &tags[i], nil
		}
	}
	return nil, errors.New(errors.ResourceNotFound, "tag not found", errors.CatAPI, false, nil)
}

func payloadErrorsToError(items []payloadError, fallback string) *errors.Error {
	for _, item := range items {
		if item.Message != "" {
			return errors.New(errors.APIError, item.Message, errors.CatAPI, false, nil)
		}
	}
	if len(items) > 0 {
		return errors.New(errors.APIError, fallback, errors.CatAPI, false, nil)
	}
	return nil
}

func (s *Service) CreateTag(ctx context.Context, name, color string) (*Tag, error) {
	var resp struct {
		CreateTransactionTag struct {
			Tag    *rawTag        `json:"tag"`
			Errors []payloadError `json:"errors"`
		} `json:"createTransactionTag"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_CreateTransactionTag",
		Query:         CreateTagMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"name":  name,
				"color": color,
			},
		},
	}, &resp)

	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.CreateTransactionTag.Errors, "failed to create tag"); apiErr != nil {
		return nil, apiErr
	}
	if resp.CreateTransactionTag.Tag == nil {
		return nil, errors.New(errors.APISchemaChanged, "tag creation response missing tag", errors.CatAPI, false, nil)
	}
	tag := toTag(resp.CreateTransactionTag.Tag)
	return &tag, nil
}

func (s *Service) UpdateTag(ctx context.Context, id string, name, color *string) (*Tag, error) {
	current, err := s.GetTag(ctx, id)
	if err != nil {
		return nil, err
	}
	newName := current.Name
	if name != nil {
		newName = *name
	}
	newColor := current.Color
	if color != nil {
		newColor = *color
	}

	var resp struct {
		UpdateTransactionTag struct {
			Tag    *rawTag        `json:"tag"`
			Errors []payloadError `json:"errors"`
		} `json:"updateTransactionTag"`
	}

	err = s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateTransactionTag",
		Query:         UpdateTagMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"id":    id,
				"name":  newName,
				"color": newColor,
			},
		},
	}, &resp)

	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.UpdateTransactionTag.Errors, "failed to update tag"); apiErr != nil {
		return nil, apiErr
	}
	if resp.UpdateTransactionTag.Tag == nil {
		return nil, errors.New(errors.APISchemaChanged, "tag update response missing tag", errors.CatAPI, false, nil)
	}
	tag := toTag(resp.UpdateTransactionTag.Tag)
	return &tag, nil
}

func (s *Service) DeleteTag(ctx context.Context, id string) error {
	var resp struct {
		DeleteTransactionTag struct {
			Errors []payloadError `json:"errors"`
		} `json:"deleteTransactionTag"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_DeleteHouseholdTransactionTag",
		Query:         DeleteTagMutation,
		Variables:     map[string]any{"tagId": id},
	}, &resp)

	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.DeleteTransactionTag.Errors, "failed to delete tag"); apiErr != nil {
		return apiErr
	}
	return nil
}

func (s *Service) ReorderTag(ctx context.Context, id string, order int) ([]Tag, error) {
	var resp struct {
		UpdateTransactionTagOrder struct {
			HouseholdTransactionTags []*rawTag `json:"householdTransactionTags"`
		} `json:"updateTransactionTagOrder"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateTransactionTagOrder",
		Query:         ReorderTagMutation,
		Variables: map[string]any{
			"tagId": id,
			"order": order,
		},
	}, &resp)

	if err != nil {
		return nil, err
	}
	tags := make([]Tag, 0, len(resp.UpdateTransactionTagOrder.HouseholdTransactionTags))
	for _, t := range resp.UpdateTransactionTagOrder.HouseholdTransactionTags {
		if t != nil {
			tags = append(tags, toTag(t))
		}
	}
	sortTags(tags)
	return tags, nil
}
