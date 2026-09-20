package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetGoalsQuery = queries.Get("goals/list.graphql")
var GetSavingsGoalBudgetsQuery = queries.Get("goals/budgets.graphql")
var GetGoalQuery = queries.Get("goals/show.graphql")
var CreateGoalMutation = queries.Get("goals/create.graphql")
var UpdateGoalMutation = queries.Get("goals/update.graphql")
var DeleteGoalMutation = queries.Get("goals/delete.graphql")
var ArchiveGoalMutation = queries.Get("goals/archive.graphql")
var UnarchiveGoalMutation = queries.Get("goals/unarchive.graphql")
var UpdateGoalPrioritiesMutation = queries.Get("goals/priorities.graphql")
var LinkGoalAccountMutation = queries.Get("goals/link_account.graphql")
var ListGoalEventsQuery = queries.Get("goals/events.graphql")
var ContributeToGoalMutation = queries.Get("goals/contribute.graphql")
var WithdrawFromGoalMutation = queries.Get("goals/withdraw.graphql")
var UpdateGoalEventMutation = queries.Get("goals/event_update.graphql")
var DeleteGoalEventMutation = queries.Get("goals/event_delete.graphql")
var GetGoalBudgetAmountsQuery = queries.Get("goals/budget_amounts.graphql")
var SetGoalBudgetAmountMutation = queries.Get("goals/budget_amount_set.graphql")

type Goal struct {
	ID                                    string  `json:"id"`
	Type                                  string  `json:"type"`
	Name                                  string  `json:"name"`
	Status                                string  `json:"status"`
	Progress                              float64 `json:"progress"`
	CurrentBalance                        float64 `json:"current_balance"`
	TargetDate                            string  `json:"target_date"`
	TargetAmount                          float64 `json:"target_amount"`
	PlannedMonthlyContribution            float64 `json:"planned_monthly_contribution"`
	CurrentMonthPlannedContributionAmount float64 `json:"current_month_planned_contribution_amount"`
	SpendingTotal                         float64 `json:"spending_total"`
	NetContribution                       float64 `json:"net_contribution"`
	EstimatedMonthsUntilCompletion        int     `json:"estimated_months_until_completion"`
	ForecastedCompletionDate              string  `json:"forecasted_completion_date"`
	IsSinkingFund                         bool    `json:"is_sinking_fund"`
	Priority                              int     `json:"priority"`
}

type SavingsGoalBudget struct {
	ID         string  `json:"id"`
	GoalID     string  `json:"goal_id"`
	GoalName   string  `json:"goal_name"`
	GoalType   string  `json:"goal_type"`
	GoalStatus string  `json:"goal_status"`
	Month      string  `json:"month"`
	Planned    float64 `json:"planned"`
	Actual     float64 `json:"actual"`
	Remaining  float64 `json:"remaining"`
}

func (s *Service) ListGoals(ctx context.Context) ([]Goal, error) {
	var resp struct {
		SavingsGoals []struct {
			ID                                    string  `json:"id"`
			Type                                  string  `json:"type"`
			Name                                  string  `json:"name"`
			Status                                string  `json:"status"`
			Progress                              float64 `json:"progress"`
			CurrentBalance                        float64 `json:"currentBalance"`
			TargetDate                            string  `json:"targetDate"`
			TargetAmount                          float64 `json:"targetAmount"`
			PlannedMonthlyContribution            float64 `json:"plannedMonthlyContribution"`
			CurrentMonthPlannedContributionAmount float64 `json:"currentMonthPlannedContributionAmount"`
			SpendingTotal                         float64 `json:"spendingTotal"`
			NetContribution                       float64 `json:"netContribution"`
			EstimatedMonthsUntilCompletion        int     `json:"estimatedMonthsUntilCompletion"`
			ForecastedCompletionDate              string  `json:"forecastedCompletionDate"`
			IsSinkingFund                         bool    `json:"isSinkingFund"`
			Priority                              int     `json:"priority"`
		} `json:"savingsGoals"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_SavingsGoals",
		Query:         GetGoalsQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}

	goals := make([]Goal, len(resp.SavingsGoals))
	for i := range resp.SavingsGoals {
		g := &resp.SavingsGoals[i]
		goals[i] = Goal{
			ID:                                    g.ID,
			Type:                                  g.Type,
			Name:                                  g.Name,
			Status:                                g.Status,
			Progress:                              g.Progress,
			CurrentBalance:                        g.CurrentBalance,
			TargetDate:                            g.TargetDate,
			TargetAmount:                          g.TargetAmount,
			PlannedMonthlyContribution:            g.PlannedMonthlyContribution,
			CurrentMonthPlannedContributionAmount: g.CurrentMonthPlannedContributionAmount,
			SpendingTotal:                         g.SpendingTotal,
			NetContribution:                       g.NetContribution,
			EstimatedMonthsUntilCompletion:        g.EstimatedMonthsUntilCompletion,
			ForecastedCompletionDate:              g.ForecastedCompletionDate,
			IsSinkingFund:                         g.IsSinkingFund,
			Priority:                              g.Priority,
		}
	}
	return goals, nil
}

func (s *Service) ListSavingsGoalBudgets(ctx context.Context, startDate, endDate string) ([]SavingsGoalBudget, error) {
	var resp struct {
		SavingsGoalMonthlyBudgetAmounts []struct {
			ID          string `json:"id"`
			SavingsGoal struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"savingsGoal"`
			MonthlyAmounts []struct {
				Month           string  `json:"month"`
				PlannedAmount   float64 `json:"plannedAmount"`
				ActualAmount    float64 `json:"actualAmount"`
				RemainingAmount float64 `json:"remainingAmount"`
			} `json:"monthlyAmounts"`
		} `json:"savingsGoalMonthlyBudgetAmounts"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetSavingsGoals",
		Query:         GetSavingsGoalBudgetsQuery,
		Variables: map[string]any{
			"startDate": startDate,
			"endDate":   endDate,
		},
	}, &resp)
	if err != nil {
		return nil, err
	}

	budgets := make([]SavingsGoalBudget, 0)
	for _, item := range resp.SavingsGoalMonthlyBudgetAmounts {
		for _, m := range item.MonthlyAmounts {
			budgets = append(budgets, SavingsGoalBudget{
				ID:         item.ID,
				GoalID:     item.SavingsGoal.ID,
				GoalName:   item.SavingsGoal.Name,
				GoalType:   item.SavingsGoal.Type,
				GoalStatus: item.SavingsGoal.Status,
				Month:      m.Month,
				Planned:    m.PlannedAmount,
				Actual:     m.ActualAmount,
				Remaining:  m.RemainingAmount,
			})
		}
	}
	return budgets, nil
}

type rawGoal struct {
	ID                                    string  `json:"id"`
	Type                                  string  `json:"type"`
	Name                                  string  `json:"name"`
	Status                                string  `json:"status"`
	Progress                              float64 `json:"progress"`
	CurrentBalance                        float64 `json:"currentBalance"`
	TargetDate                            string  `json:"targetDate"`
	TargetAmount                          float64 `json:"targetAmount"`
	PlannedMonthlyContribution            float64 `json:"plannedMonthlyContribution"`
	CurrentMonthPlannedContributionAmount float64 `json:"currentMonthPlannedContributionAmount"`
	SpendingTotal                         float64 `json:"spendingTotal"`
	NetContribution                       float64 `json:"netContribution"`
	EstimatedMonthsUntilCompletion        int     `json:"estimatedMonthsUntilCompletion"`
	ForecastedCompletionDate              string  `json:"forecastedCompletionDate"`
	IsSinkingFund                         bool    `json:"isSinkingFund"`
	Priority                              int     `json:"priority"`
}

func toGoal(g *rawGoal) *Goal {
	return &Goal{
		ID:                                    g.ID,
		Type:                                  g.Type,
		Name:                                  g.Name,
		Status:                                g.Status,
		Progress:                              g.Progress,
		CurrentBalance:                        g.CurrentBalance,
		TargetDate:                            g.TargetDate,
		TargetAmount:                          g.TargetAmount,
		PlannedMonthlyContribution:            g.PlannedMonthlyContribution,
		CurrentMonthPlannedContributionAmount: g.CurrentMonthPlannedContributionAmount,
		SpendingTotal:                         g.SpendingTotal,
		NetContribution:                       g.NetContribution,
		EstimatedMonthsUntilCompletion:        g.EstimatedMonthsUntilCompletion,
		ForecastedCompletionDate:              g.ForecastedCompletionDate,
		IsSinkingFund:                         g.IsSinkingFund,
		Priority:                              g.Priority,
	}
}

func (s *Service) GetGoal(ctx context.Context, id string) (*Goal, error) {
	var resp struct {
		SavingsGoal *rawGoal `json:"savingsGoal"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_SavingsGoal",
		Query:         GetGoalQuery,
		Variables:     map[string]any{"id": id},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.SavingsGoal == nil {
		return nil, errors.New(errors.ResourceNotFound, "goal not found", errors.CatAPI, false, nil)
	}
	return toGoal(resp.SavingsGoal), nil
}

type GoalInput struct {
	Name                string
	Type                string
	TargetAmount        *float64
	TargetDate          *string
	MonthlyContribution *float64
	SinkingFund         *bool
	Priority            *int
}

func (s *Service) CreateGoal(ctx context.Context, input *GoalInput) (*Goal, error) {
	goal := map[string]any{"name": input.Name}
	if input.Type != "" {
		goal["type"] = input.Type
	}
	if input.TargetAmount != nil {
		goal["targetAmount"] = *input.TargetAmount
	}
	if input.TargetDate != nil {
		goal["targetDate"] = *input.TargetDate
	}
	if input.MonthlyContribution != nil {
		goal["plannedMonthlyContribution"] = *input.MonthlyContribution
	}
	if input.SinkingFund != nil {
		goal["isSinkingFund"] = *input.SinkingFund
	}
	if input.Priority != nil {
		goal["priority"] = *input.Priority
	}

	var resp struct {
		CreateSavingsGoals struct {
			SavingsGoals []struct {
				ID string `json:"id"`
			} `json:"savingsGoals"`
			Errors []payloadError `json:"errors"`
		} `json:"createSavingsGoals"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_CreateSavingsGoals",
		Query:         CreateGoalMutation,
		Variables:     map[string]any{"input": map[string]any{"goals": []any{goal}}},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.CreateSavingsGoals.Errors, "failed to create goal"); apiErr != nil {
		return nil, apiErr
	}
	if len(resp.CreateSavingsGoals.SavingsGoals) == 0 {
		return nil, errors.New(errors.APISchemaChanged, "goal creation response missing savingsGoals", errors.CatAPI, false, nil)
	}
	return s.GetGoal(ctx, resp.CreateSavingsGoals.SavingsGoals[0].ID)
}

type GoalUpdate struct {
	Name                *string
	Type                *string
	TargetAmount        *float64
	TargetDate          *string
	MonthlyContribution *float64
	SinkingFund         *bool
	Priority            *int
}

func (s *Service) UpdateGoal(ctx context.Context, id string, update *GoalUpdate) (*Goal, error) {
	input := map[string]any{"id": id}
	if update.Name != nil {
		input["name"] = *update.Name
	}
	if update.Type != nil {
		input["type"] = *update.Type
	}
	if update.TargetAmount != nil {
		input["targetAmount"] = *update.TargetAmount
	}
	if update.TargetDate != nil {
		input["targetDate"] = *update.TargetDate
	}
	if update.MonthlyContribution != nil {
		input["budgetToApplyToFutureMonths"] = *update.MonthlyContribution
	}
	if update.SinkingFund != nil {
		input["isSinkingFund"] = *update.SinkingFund
	}
	if update.Priority != nil {
		input["priority"] = *update.Priority
	}

	var resp struct {
		UpdateSavingsGoal struct {
			SavingsGoal *rawGoal       `json:"savingsGoal"`
			Errors      []payloadError `json:"errors"`
		} `json:"updateSavingsGoal"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateSavingsGoal",
		Query:         UpdateGoalMutation,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.UpdateSavingsGoal.Errors, "failed to update goal"); apiErr != nil {
		return nil, apiErr
	}
	if resp.UpdateSavingsGoal.SavingsGoal == nil {
		return nil, errors.New(errors.APISchemaChanged, "goal update response missing savingsGoal", errors.CatAPI, false, nil)
	}
	return toGoal(resp.UpdateSavingsGoal.SavingsGoal), nil
}

func (s *Service) DeleteGoal(ctx context.Context, id string) error {
	var resp struct {
		DeleteSavingsGoal struct {
			Success bool           `json:"success"`
			Errors  []payloadError `json:"errors"`
		} `json:"deleteSavingsGoal"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_DeleteSavingsGoal",
		Query:         DeleteGoalMutation,
		Variables:     map[string]any{"input": map[string]any{"id": id}},
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.DeleteSavingsGoal.Errors, "failed to delete goal"); apiErr != nil {
		return apiErr
	}
	if !resp.DeleteSavingsGoal.Success {
		return errors.New(errors.APIError, "failed to delete goal", errors.CatAPI, false, nil)
	}
	return nil
}

func (s *Service) ArchiveGoal(ctx context.Context, id string) (*Goal, error) {
	var resp struct {
		ArchiveSavingsGoal struct {
			SavingsGoal *rawGoal       `json:"savingsGoal"`
			Errors      []payloadError `json:"errors"`
		} `json:"archiveSavingsGoal"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_ArchiveSavingsGoal",
		Query:         ArchiveGoalMutation,
		Variables:     map[string]any{"input": map[string]any{"id": id}},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.ArchiveSavingsGoal.Errors, "failed to archive goal"); apiErr != nil {
		return nil, apiErr
	}
	if resp.ArchiveSavingsGoal.SavingsGoal == nil {
		return nil, errors.New(errors.APISchemaChanged, "goal archive response missing savingsGoal", errors.CatAPI, false, nil)
	}
	return toGoal(resp.ArchiveSavingsGoal.SavingsGoal), nil
}

func (s *Service) RestoreGoal(ctx context.Context, id string) (*Goal, error) {
	var resp struct {
		UnarchiveSavingsGoal struct {
			SavingsGoal *rawGoal       `json:"savingsGoal"`
			Errors      []payloadError `json:"errors"`
		} `json:"unarchiveSavingsGoal"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UnarchiveSavingsGoal",
		Query:         UnarchiveGoalMutation,
		Variables:     map[string]any{"input": map[string]any{"id": id}},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.UnarchiveSavingsGoal.Errors, "failed to restore goal"); apiErr != nil {
		return nil, apiErr
	}
	if resp.UnarchiveSavingsGoal.SavingsGoal == nil {
		return nil, errors.New(errors.APISchemaChanged, "goal restore response missing savingsGoal", errors.CatAPI, false, nil)
	}
	return toGoal(resp.UnarchiveSavingsGoal.SavingsGoal), nil
}

type GoalEvent struct {
	ID          string  `json:"id"`
	Date        string  `json:"date,omitempty"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type,omitempty"`
	Notes       string  `json:"notes,omitempty"`
	GoalID      string  `json:"goal_id,omitempty"`
	GoalName    string  `json:"goal_name,omitempty"`
	AccountID   string  `json:"account_id,omitempty"`
	AccountName string  `json:"account_name,omitempty"`
}

type GoalBudgetAmount struct {
	ID        string  `json:"id"`
	Month     string  `json:"month"`
	Planned   float64 `json:"planned"`
	Actual    float64 `json:"actual"`
	Remaining float64 `json:"remaining"`
}

type rawGoalEvent struct {
	ID     string  `json:"id"`
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Type   string  `json:"type"`
	Notes  string  `json:"notes"`
	Goal   *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"goal"`
	Account *struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"account"`
}

func toGoalEvent(r *rawGoalEvent) *GoalEvent {
	e := &GoalEvent{ID: r.ID, Date: r.Date, Amount: r.Amount, Type: r.Type, Notes: r.Notes}
	if r.Goal != nil {
		e.GoalID, e.GoalName = r.Goal.ID, r.Goal.Name
	}
	if r.Account != nil {
		e.AccountID, e.AccountName = r.Account.ID, r.Account.DisplayName
	}
	return e
}

func (s *Service) UpdateGoalPriorities(ctx context.Context, ids []string) error {
	goals := make([]map[string]any, len(ids))
	for i, id := range ids {
		goals[i] = map[string]any{"id": id, "priority": i}
	}
	var resp struct {
		UpdateSavingsGoalsPriorities struct {
			Errors []payloadError `json:"errors"`
		} `json:"updateSavingsGoalsPriorities"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateSavingsGoalsPriorities",
		Query:         UpdateGoalPrioritiesMutation,
		Variables:     map[string]any{"input": map[string]any{"goals": goals}},
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.UpdateSavingsGoalsPriorities.Errors, "failed to update goal priorities"); apiErr != nil {
		return apiErr
	}
	return nil
}

func (s *Service) setGoalAccountContribution(ctx context.Context, goalID, accountID string, amount *float64, useEntireBalance bool) (*Goal, error) {
	contributed := map[string]any{
		"goalId":                      goalID,
		"overrideInitialContribution": true,
		"useEntireBalance":            useEntireBalance,
	}
	if amount != nil {
		contributed["contributionAmount"] = *amount
	}
	var resp struct {
		CreateGoalAccountInitialContributions struct {
			Errors []payloadError `json:"errors"`
		} `json:"createGoalAccountInitialContributions"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_CreateSavingsGoalAccountInitialContributions",
		Query:         LinkGoalAccountMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"accountId":        accountID,
				"contributedGoals": []any{contributed},
			},
		},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.CreateGoalAccountInitialContributions.Errors, "failed to link goal account"); apiErr != nil {
		return nil, apiErr
	}
	return s.GetGoal(ctx, goalID)
}

func (s *Service) LinkGoalAccount(ctx context.Context, goalID, accountID string, amount *float64, useEntireBalance bool) (*Goal, error) {
	return s.setGoalAccountContribution(ctx, goalID, accountID, amount, useEntireBalance)
}

func (s *Service) UnlinkGoalAccount(ctx context.Context, goalID, accountID string) (*Goal, error) {
	return s.setGoalAccountContribution(ctx, goalID, accountID, nil, false)
}

func (s *Service) ListGoalEvents(ctx context.Context, goalID string) ([]*GoalEvent, error) {
	var resp struct {
		SavingsGoal *struct {
			GoalEvents []*rawGoalEvent `json:"goalEvents"`
		} `json:"savingsGoal"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_SavingsGoalEvents",
		Query:         ListGoalEventsQuery,
		Variables:     map[string]any{"id": goalID},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.SavingsGoal == nil {
		return []*GoalEvent{}, nil
	}
	out := make([]*GoalEvent, 0, len(resp.SavingsGoal.GoalEvents))
	for _, r := range resp.SavingsGoal.GoalEvents {
		if r != nil {
			out = append(out, toGoalEvent(r))
		}
	}
	return out, nil
}

func (s *Service) createGoalEvent(ctx context.Context, operation, query, goalID, accountID string, amount float64, date, notes *string) (*GoalEvent, error) {
	input := map[string]any{
		"id": goalID, "accountId": accountID, "amount": amount,
	}
	if date != nil {
		input["date"] = *date
	}
	if notes != nil {
		input["notes"] = *notes
	}
	var raw struct {
		CreateSavingsGoalContribution *struct {
			GoalEvent *rawGoalEvent `json:"goalEvent"`
		} `json:"createSavingsGoalContribution"`
		CreateSavingsGoalWithdrawal *struct {
			GoalEvent *rawGoalEvent `json:"goalEvent"`
		} `json:"createSavingsGoalWithdrawal"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: operation,
		Query:         query,
		Variables:     map[string]any{"input": input},
	}, &raw)
	if err != nil {
		return nil, err
	}
	var created *rawGoalEvent
	if raw.CreateSavingsGoalContribution != nil {
		created = raw.CreateSavingsGoalContribution.GoalEvent
	}
	if raw.CreateSavingsGoalWithdrawal != nil {
		created = raw.CreateSavingsGoalWithdrawal.GoalEvent
	}
	if created == nil || created.ID == "" {
		return nil, errors.New(errors.APISchemaChanged, "goal event creation response missing goalEvent", errors.CatAPI, false, nil)
	}
	events, err := s.ListGoalEvents(ctx, goalID)
	if err != nil {
		return nil, err
	}
	for _, e := range events {
		if e.ID == created.ID {
			return e, nil
		}
	}
	return nil, errors.New(errors.APISchemaChanged, "goal event was created but could not be refetched", errors.CatAPI, false, nil)
}

func (s *Service) ContributeToGoal(ctx context.Context, goalID, accountID string, amount float64, date, notes *string) (*GoalEvent, error) {
	return s.createGoalEvent(ctx, "Common_ContributeToSavingsGoal", ContributeToGoalMutation, goalID, accountID, amount, date, notes)
}

func (s *Service) WithdrawFromGoal(ctx context.Context, goalID, accountID string, amount float64, date, notes *string) (*GoalEvent, error) {
	return s.createGoalEvent(ctx, "Common_WithdrawFromSavingsGoal", WithdrawFromGoalMutation, goalID, accountID, amount, date, notes)
}

func (s *Service) UpdateGoalEvent(ctx context.Context, eventID string, date, notes *string) (*GoalEvent, error) {
	input := map[string]any{"eventId": eventID}
	if date != nil {
		input["date"] = *date
	}
	if notes != nil {
		input["notes"] = *notes
	}
	var resp struct {
		UpdateGoalEvent struct {
			GoalEvent *rawGoalEvent `json:"goalEvent"`
		} `json:"updateGoalEvent"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateSavingsGoalEvent",
		Query:         UpdateGoalEventMutation,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.UpdateGoalEvent.GoalEvent == nil {
		return nil, errors.New(errors.APISchemaChanged, "goal event update response missing goalEvent", errors.CatAPI, false, nil)
	}
	return toGoalEvent(resp.UpdateGoalEvent.GoalEvent), nil
}

func (s *Service) DeleteGoalEvent(ctx context.Context, eventID string) error {
	var resp struct {
		DeleteGoalEvent struct {
			Success bool `json:"success"`
		} `json:"deleteGoalEvent"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_DeleteSavingsGoalEvent",
		Query:         DeleteGoalEventMutation,
		Variables:     map[string]any{"input": map[string]any{"eventId": eventID}},
	}, &resp)
	if err != nil {
		return err
	}
	if !resp.DeleteGoalEvent.Success {
		return errors.New(errors.APIError, "failed to delete goal event", errors.CatAPI, false, nil)
	}
	return nil
}

func (s *Service) GetGoalBudgetAmounts(ctx context.Context, goalID, startMonth, endMonth string) ([]GoalBudgetAmount, error) {
	var resp struct {
		SavingsGoal *struct {
			MonthlyBudgetAmounts []struct {
				ID              string  `json:"id"`
				Month           string  `json:"month"`
				PlannedAmount   float64 `json:"plannedAmount"`
				ActualAmount    float64 `json:"actualAmount"`
				RemainingAmount float64 `json:"remainingAmount"`
			} `json:"monthlyBudgetAmounts"`
		} `json:"savingsGoal"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_SavingsGoalBudgetAmounts",
		Query:         GetGoalBudgetAmountsQuery,
		Variables: map[string]any{
			"goalId": goalID, "startMonth": startMonth, "endMonth": endMonth,
		},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.SavingsGoal == nil {
		return []GoalBudgetAmount{}, nil
	}
	out := make([]GoalBudgetAmount, 0, len(resp.SavingsGoal.MonthlyBudgetAmounts))
	for _, m := range resp.SavingsGoal.MonthlyBudgetAmounts {
		out = append(out, GoalBudgetAmount{
			ID: m.ID, Month: m.Month,
			Planned: m.PlannedAmount, Actual: m.ActualAmount, Remaining: m.RemainingAmount,
		})
	}
	return out, nil
}

func (s *Service) SetGoalBudgetAmount(ctx context.Context, goalID, month string, amount float64, applyToFuture bool, accountID string) error {
	input := map[string]any{
		"savingsGoalId": goalID, "month": month,
		"amount": amount, "applyToFuture": applyToFuture,
	}
	if accountID != "" {
		input["accountId"] = accountID
	}
	var resp struct {
		SetSavingsGoalBudgetAmount struct {
			Success bool           `json:"success"`
			Errors  []payloadError `json:"errors"`
		} `json:"setSavingsGoalBudgetAmount"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_SetSavingsGoalBudgetAmount",
		Query:         SetGoalBudgetAmountMutation,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.SetSavingsGoalBudgetAmount.Errors, "failed to set goal budget amount"); apiErr != nil {
		return apiErr
	}
	if !resp.SetSavingsGoalBudgetAmount.Success {
		return errors.New(errors.APIError, "failed to set goal budget amount", errors.CatAPI, false, nil)
	}
	return nil
}
