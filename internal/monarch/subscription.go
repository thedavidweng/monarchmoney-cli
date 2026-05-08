package monarch

import (
	"context"
	_ "embed"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

//go:embed queries/subscription/show.graphql
var GetSubscriptionQuery string

type Subscription struct {
	Status           string `json:"status"`
	PlanName         string `json:"plan_name"`
	CurrentPeriodEnd string `json:"current_period_end"`
}

func (s *Service) GetSubscriptionDetails(ctx context.Context) (*Subscription, error) {
	var resp struct {
		Subscription struct {
			Status string `json:"status"`
			Plan   struct {
				Name string `json:"name"`
			} `json:"plan"`
			CurrentPeriodEnd string `json:"currentPeriodEnd"`
		} `json:"subscription"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetSubscriptionDetails",
		Query:         GetSubscriptionQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &Subscription{
		Status:           resp.Subscription.Status,
		PlanName:         resp.Subscription.Plan.Name,
		CurrentPeriodEnd: resp.Subscription.CurrentPeriodEnd,
	}, nil
}
