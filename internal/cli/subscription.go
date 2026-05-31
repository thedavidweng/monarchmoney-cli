package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

var subscriptionCmd = &cobra.Command{
	Use:   "subscription",
	Short: "Manage subscription details",
}

var subscriptionShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show subscription details",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "subscription.show", start)
		if !ok {
			return
		}
		svc := deps.Service

		sub, err := svc.GetSubscriptionDetails(cmd.Context())
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get subscription details", errors.CatAPI, false, err)
			}
			handleError(renderer, "subscription.show", cliErr, start)
			return
		}

		if jsonMode {
			env := envelopeWithWarnings("subscription.show", sub, start, "uses legacy Monarch GraphQL root field: subscription")
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("ID:                      %s\n", sub.ID)
			fmt.Printf("Payment Source:          %s\n", sub.PaymentSource)
			fmt.Printf("Referral Code:           %s\n", sub.ReferralCode)
			fmt.Printf("On Free Trial:           %v\n", sub.IsOnFreeTrial)
			fmt.Printf("Has Premium Entitlement: %v\n", sub.HasPremiumEntitlement)
		}
	},
}

func init() {
	subscriptionCmd.AddCommand(subscriptionShowCmd)
	RootCmd.AddCommand(subscriptionCmd)
}
