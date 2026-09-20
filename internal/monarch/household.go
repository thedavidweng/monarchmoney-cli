package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetHouseholdQuery = queries.Get("household/show.graphql")
var ListHouseholdMembersQuery = queries.Get("household/members.graphql")
var GetCurrentUserQuery = queries.Get("household/me.graphql")
var UpdateCurrentUserMutation = queries.Get("household/update_me.graphql")
var GetHouseholdPreferencesQuery = queries.Get("household/preferences.graphql")
var UpdateHouseholdPreferencesMutation = queries.Get("household/update_preferences.graphql")

type Household struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address,omitempty"`
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`
	ZipCode string `json:"zip_code,omitempty"`
	Country string `json:"country,omitempty"`
}

type HouseholdMember struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	HasMFA      *bool  `json:"has_mfa,omitempty"`
}

type UserProfile struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Timezone    string `json:"timezone"`
	Role        string `json:"role"`
	HasMFA      *bool  `json:"has_mfa,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

type HouseholdPreferences struct {
	ID                            string `json:"id"`
	NewTransactionsNeedReview     *bool  `json:"new_transactions_need_review,omitempty"`
	UncategorizedNeedReview       *bool  `json:"uncategorized_transactions_need_review,omitempty"`
	PendingCanBeEdited            *bool  `json:"pending_transactions_can_be_edited,omitempty"`
	AIAssistantEnabled            *bool  `json:"ai_assistant_enabled,omitempty"`
	InvestmentTransactionsEnabled *bool  `json:"investment_transactions_enabled,omitempty"`
	ApplyToFutureMonthsDefault    *bool  `json:"budget_apply_to_future_months_default,omitempty"`
	HiddenTransactionsBetaEnabled *bool  `json:"hidden_transactions_beta_enabled,omitempty"`
	ExcludeBusinessFromBudget     *bool  `json:"exclude_business_from_budget,omitempty"`
	EligibleForFinancialInsights  *bool  `json:"eligible_for_financial_insights,omitempty"`
	BudgetSystem                  string `json:"budget_system,omitempty"`
}

func (s *Service) GetHousehold(ctx context.Context) (*Household, error) {
	var resp struct {
		MyHousehold *struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Address string `json:"address"`
			City    string `json:"city"`
			State   string `json:"state"`
			ZipCode string `json:"zipCode"`
			Country string `json:"country"`
		} `json:"myHousehold"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_GetMyHousehold",
		Query:         GetHouseholdQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.MyHousehold == nil {
		return nil, errors.New(errors.APISchemaChanged, "household response missing myHousehold", errors.CatAPI, false, nil)
	}
	h := resp.MyHousehold
	return &Household{ID: h.ID, Name: h.Name, Address: h.Address, City: h.City, State: h.State, ZipCode: h.ZipCode, Country: h.Country}, nil
}

func (s *Service) ListHouseholdMembers(ctx context.Context) ([]HouseholdMember, error) {
	var resp struct {
		MyHousehold *struct {
			Users []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				DisplayName string `json:"displayName"`
				Email       string `json:"email"`
				Role        string `json:"householdRole"`
				HasMFA      *bool  `json:"hasMfaOn"`
			} `json:"users"`
		} `json:"myHousehold"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_GetHouseholdMembers",
		Query:         ListHouseholdMembersQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.MyHousehold == nil {
		return []HouseholdMember{}, nil
	}
	out := make([]HouseholdMember, 0, len(resp.MyHousehold.Users))
	for _, u := range resp.MyHousehold.Users {
		out = append(out, HouseholdMember{ID: u.ID, Name: u.Name, DisplayName: u.DisplayName, Email: u.Email, Role: u.Role, HasMFA: u.HasMFA})
	}
	return out, nil
}

func (s *Service) GetHouseholdMember(ctx context.Context, id string) (*HouseholdMember, error) {
	members, err := s.ListHouseholdMembers(ctx)
	if err != nil {
		return nil, err
	}
	for i := range members {
		if members[i].ID == id {
			return &members[i], nil
		}
	}
	return nil, errors.New(errors.ResourceNotFound, "household member not found", errors.CatAPI, false, nil)
}

func (s *Service) GetCurrentUser(ctx context.Context) (*UserProfile, error) {
	var resp struct {
		Me *struct {
			ID          string `json:"id"`
			Email       string `json:"email"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			Timezone    string `json:"timezone"`
			Role        string `json:"householdRole"`
			HasMFA      *bool  `json:"hasMfaOn"`
			CreatedAt   string `json:"createdAt"`
		} `json:"me"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_GetMe",
		Query:         GetCurrentUserQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Me == nil {
		return nil, errors.New(errors.APISchemaChanged, "identity response missing me", errors.CatAPI, false, nil)
	}
	u := resp.Me
	return &UserProfile{ID: u.ID, Email: u.Email, Name: u.Name, DisplayName: u.DisplayName, Timezone: u.Timezone, Role: u.Role, HasMFA: u.HasMFA, CreatedAt: u.CreatedAt}, nil
}

func (s *Service) UpdateCurrentUser(ctx context.Context, displayName, timezone *string) (*UserProfile, error) {
	input := map[string]any{}
	if displayName != nil {
		input["displayName"] = *displayName
	}
	if timezone != nil {
		input["timezone"] = *timezone
	}
	if len(input) == 0 {
		return s.GetCurrentUser(ctx)
	}

	var resp struct {
		UpdateMe struct {
			User *struct {
				ID          string `json:"id"`
				Email       string `json:"email"`
				Name        string `json:"name"`
				DisplayName string `json:"displayName"`
				Timezone    string `json:"timezone"`
				Role        string `json:"householdRole"`
				HasMFA      *bool  `json:"hasMfaOn"`
				CreatedAt   string `json:"createdAt"`
			} `json:"user"`
			Errors []payloadError `json:"errors"`
		} `json:"updateMe"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateMe",
		Query:         UpdateCurrentUserMutation,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.UpdateMe.Errors, "failed to update user"); apiErr != nil {
		return nil, apiErr
	}
	if resp.UpdateMe.User == nil {
		return nil, errors.New(errors.APISchemaChanged, "user update response missing user", errors.CatAPI, false, nil)
	}
	u := resp.UpdateMe.User
	return &UserProfile{ID: u.ID, Email: u.Email, Name: u.Name, DisplayName: u.DisplayName, Timezone: u.Timezone, Role: u.Role, HasMFA: u.HasMFA, CreatedAt: u.CreatedAt}, nil
}

func (s *Service) GetHouseholdPreferences(ctx context.Context) (*HouseholdPreferences, error) {
	var resp struct {
		HouseholdPreferences *struct {
			ID                            string `json:"id"`
			NewTransactionsNeedReview     *bool  `json:"newTransactionsNeedReview"`
			UncategorizedNeedReview       *bool  `json:"uncategorizedTransactionsNeedReview"`
			PendingCanBeEdited            *bool  `json:"pendingTransactionsCanBeEdited"`
			AIAssistantEnabled            *bool  `json:"aiAssistantEnabled"`
			InvestmentTransactionsEnabled *bool  `json:"investmentTransactionsEnabled"`
			ApplyToFutureMonthsDefault    *bool  `json:"budgetApplyToFutureMonthsDefault"`
			HiddenBetaEnabled             *bool  `json:"hiddenTransactionsBetaEnabled"`
			ExcludeBusinessFromBudget     *bool  `json:"excludeBusinessFromBudget"`
			EligibleForInsights           *bool  `json:"eligibleForFinancialInsights"`
		} `json:"householdPreferences"`
		BudgetSystem string `json:"budgetSystem"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_GetHouseholdPreferences",
		Query:         GetHouseholdPreferencesQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.HouseholdPreferences == nil {
		return nil, errors.New(errors.APISchemaChanged, "preferences response missing householdPreferences", errors.CatAPI, false, nil)
	}
	p := resp.HouseholdPreferences
	return &HouseholdPreferences{
		ID:                            p.ID,
		NewTransactionsNeedReview:     p.NewTransactionsNeedReview,
		UncategorizedNeedReview:       p.UncategorizedNeedReview,
		PendingCanBeEdited:            p.PendingCanBeEdited,
		AIAssistantEnabled:            p.AIAssistantEnabled,
		InvestmentTransactionsEnabled: p.InvestmentTransactionsEnabled,
		ApplyToFutureMonthsDefault:    p.ApplyToFutureMonthsDefault,
		HiddenTransactionsBetaEnabled: p.HiddenBetaEnabled,
		ExcludeBusinessFromBudget:     p.ExcludeBusinessFromBudget,
		EligibleForFinancialInsights:  p.EligibleForInsights,
		BudgetSystem:                  resp.BudgetSystem,
	}, nil
}

type HouseholdPreferencesUpdate struct {
	NewTransactionsNeedReview     *bool
	UncategorizedNeedReview       *bool
	PendingCanBeEdited            *bool
	HiddenTransactionsBetaEnabled *bool
	ExcludeBusinessFromBudget     *bool
}

func (s *Service) UpdateHouseholdPreferences(ctx context.Context, update *HouseholdPreferencesUpdate) (*HouseholdPreferences, error) {
	input := map[string]any{}
	if update.NewTransactionsNeedReview != nil {
		input["newTransactionsNeedReview"] = *update.NewTransactionsNeedReview
	}
	if update.UncategorizedNeedReview != nil {
		input["uncategorizedTransactionsNeedReview"] = *update.UncategorizedNeedReview
	}
	if update.PendingCanBeEdited != nil {
		input["pendingTransactionsCanBeEdited"] = *update.PendingCanBeEdited
	}
	if update.HiddenTransactionsBetaEnabled != nil {
		input["hiddenTransactionsBetaEnabled"] = *update.HiddenTransactionsBetaEnabled
	}
	if update.ExcludeBusinessFromBudget != nil {
		input["excludeBusinessFromBudget"] = *update.ExcludeBusinessFromBudget
	}
	if len(input) == 0 {
		return s.GetHouseholdPreferences(ctx)
	}

	var resp struct {
		UpdateHouseholdPreferences struct {
			HouseholdPreferences *struct {
				ID string `json:"id"`
			} `json:"householdPreferences"`
		} `json:"updateHouseholdPreferences"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateHouseholdPreferences",
		Query:         UpdateHouseholdPreferencesMutation,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.UpdateHouseholdPreferences.HouseholdPreferences == nil {
		return nil, errors.New(errors.APISchemaChanged, "preferences update response missing householdPreferences", errors.CatAPI, false, nil)
	}
	return s.GetHouseholdPreferences(ctx)
}
