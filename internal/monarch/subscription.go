package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetSubscriptionQuery = queries.Get("subscription/show.graphql")

type Subscription struct {
	ID                    string `json:"id"`
	PaymentSource         string `json:"payment_source"`
	ReferralCode          string `json:"referral_code"`
	IsOnFreeTrial         bool   `json:"is_on_free_trial"`
	HasPremiumEntitlement bool   `json:"has_premium_entitlement"`
}

func (s *Service) GetSubscriptionDetails(ctx context.Context) (*Subscription, error) {
	var resp struct {
		Subscription struct {
			ID                    string `json:"id"`
			PaymentSource         string `json:"paymentSource"`
			ReferralCode          string `json:"referralCode"`
			IsOnFreeTrial         bool   `json:"isOnFreeTrial"`
			HasPremiumEntitlement bool   `json:"hasPremiumEntitlement"`
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
		ID:                    resp.Subscription.ID,
		PaymentSource:         resp.Subscription.PaymentSource,
		ReferralCode:          resp.Subscription.ReferralCode,
		IsOnFreeTrial:         resp.Subscription.IsOnFreeTrial,
		HasPremiumEntitlement: resp.Subscription.HasPremiumEntitlement,
	}, nil
}
