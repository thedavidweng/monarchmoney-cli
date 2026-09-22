package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetDebtPaydownQuery = queries.Get("debt/paydown.graphql")

type DebtAccount struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Balance          float64  `json:"balance"`
	APR              *float64 `json:"apr,omitempty"`
	MinimumPayment   *float64 `json:"minimum_payment,omitempty"`
	PlannedPayment   *float64 `json:"planned_payment,omitempty"`
	ExcludedFromPlan bool     `json:"excluded_from_plan"`
}

type DebtAccountProjection struct {
	AccountID         string  `json:"account_id"`
	AccountName       string  `json:"account_name"`
	Principal         float64 `json:"principal"`
	ProjectedInterest float64 `json:"projected_interest"`
	ProjectedTotal    float64 `json:"projected_total"`
	DebtFreeDate      string  `json:"debt_free_date,omitempty"`
}

type DebtPaydownPlan struct {
	Method                    string                  `json:"method"`
	CurrentPrincipal          float64                 `json:"current_principal"`
	ProjectedInterest         float64                 `json:"projected_interest"`
	ProjectedTotal            float64                 `json:"projected_total"`
	DebtFreeDate              string                  `json:"debt_free_date,omitempty"`
	AdjustedDebtFreeDate      string                  `json:"adjusted_debt_free_date,omitempty"`
	AdjustedProjectedInterest float64                 `json:"adjusted_projected_interest"`
	AdjustedProjectedTotal    float64                 `json:"adjusted_projected_total"`
	IncludedAccounts          []DebtAccount           `json:"included_accounts"`
	ExcludedAccounts          []DebtAccount           `json:"excluded_accounts"`
	Projections               []DebtAccountProjection `json:"projections"`
}

func (s *Service) GetDebtPaydown(ctx context.Context, method string) (*DebtPaydownPlan, error) {
	if method == "" {
		method = "planned"
	}
	var resp struct {
		DebtAccounts []struct {
			ID                     string   `json:"id"`
			DisplayName            string   `json:"displayName"`
			DisplayBalance         float64  `json:"displayBalance"`
			APR                    *float64 `json:"apr"`
			MinimumPayment         *float64 `json:"minimumPayment"`
			PlannedPayment         *float64 `json:"plannedPayment"`
			ExcludeFromDebtPaydown bool     `json:"excludeFromDebtPaydown"`
		} `json:"debtAccounts"`
		DebtPaydownPlan *struct {
			CurrentDebtPrincipal      float64 `json:"currentDebtPrincipal"`
			ProjectedInterest         float64 `json:"projectedInterest"`
			ProjectedTotal            float64 `json:"projectedTotal"`
			DebtFreeDate              string  `json:"debtFreeDate"`
			AdjustedDebtFreeDate      string  `json:"adjustedDebtFreeDate"`
			AdjustedProjectedInterest float64 `json:"adjustedProjectedInterest"`
			AdjustedProjectedTotal    float64 `json:"adjustedProjectedTotal"`
			DebtAccountProjections    []struct {
				Account struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
				} `json:"account"`
				Principal         float64 `json:"principal"`
				ProjectedInterest float64 `json:"projectedInterest"`
				ProjectedTotal    float64 `json:"projectedTotal"`
				DebtFreeDate      string  `json:"debtFreeDate"`
			} `json:"debtAccountProjections"`
		} `json:"debtPaydownPlan"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetDebtPaydown",
		Query:         GetDebtPaydownQuery,
		Variables:     map[string]any{"input": map[string]any{"debtPaydownMethod": method}},
	}, &resp)
	if err != nil {
		return nil, err
	}

	plan := &DebtPaydownPlan{Method: method}
	for _, a := range resp.DebtAccounts {
		account := DebtAccount{
			ID: a.ID, Name: a.DisplayName, Balance: a.DisplayBalance,
			APR: a.APR, MinimumPayment: a.MinimumPayment, PlannedPayment: a.PlannedPayment,
			ExcludedFromPlan: a.ExcludeFromDebtPaydown,
		}
		if a.ExcludeFromDebtPaydown {
			plan.ExcludedAccounts = append(plan.ExcludedAccounts, account)
		} else {
			plan.IncludedAccounts = append(plan.IncludedAccounts, account)
		}
	}
	if plan.IncludedAccounts == nil {
		plan.IncludedAccounts = []DebtAccount{}
	}
	if plan.ExcludedAccounts == nil {
		plan.ExcludedAccounts = []DebtAccount{}
	}
	if p := resp.DebtPaydownPlan; p != nil {
		plan.CurrentPrincipal = p.CurrentDebtPrincipal
		plan.ProjectedInterest = p.ProjectedInterest
		plan.ProjectedTotal = p.ProjectedTotal
		plan.DebtFreeDate = p.DebtFreeDate
		plan.AdjustedDebtFreeDate = p.AdjustedDebtFreeDate
		plan.AdjustedProjectedInterest = p.AdjustedProjectedInterest
		plan.AdjustedProjectedTotal = p.AdjustedProjectedTotal
		for _, pr := range p.DebtAccountProjections {
			plan.Projections = append(plan.Projections, DebtAccountProjection{
				AccountID: pr.Account.ID, AccountName: pr.Account.DisplayName,
				Principal: pr.Principal, ProjectedInterest: pr.ProjectedInterest,
				ProjectedTotal: pr.ProjectedTotal, DebtFreeDate: pr.DebtFreeDate,
			})
		}
	}
	if plan.Projections == nil {
		plan.Projections = []DebtAccountProjection{}
	}
	return plan, nil
}
