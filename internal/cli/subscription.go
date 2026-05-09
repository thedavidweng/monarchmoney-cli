package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/auth"
	"github.com/thedavidweng/monarchmoney-cli/internal/config"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
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

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "subscription.show", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

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
			env := output.NewEnvelope("subscription.show", profile, output.SchemaVersion, "", sub, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Status:             %s\n", sub.Status)
			fmt.Printf("Plan:               %s\n", sub.PlanName)
			fmt.Printf("Current Period End: %s\n", sub.CurrentPeriodEnd)
		}
	},
}

func init() {
	subscriptionCmd.AddCommand(subscriptionShowCmd)
	RootCmd.AddCommand(subscriptionCmd)
}
