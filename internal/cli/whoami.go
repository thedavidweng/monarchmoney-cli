package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
)

var whoamiCmd = &cobra.Command{
	Use:     "whoami",
	Short:   "Show the signed-in user, subscription, and plan capabilities",
	GroupID: "utility",
	Example: "  monarch whoami --json",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "whoami", "failed to get identity",
			func(ctx context.Context, svc *monarch.Service) (*monarch.WhoAmI, error) {
				return svc.GetWhoAmI(ctx)
			},
			func(w *monarch.WhoAmI) {
				fmt.Printf("User:         %s <%s>\n", w.User.Name, w.User.Email)
				fmt.Printf("Timezone:     %s\n", w.User.Timezone)
				fmt.Printf("Premium:      %v\n", w.Subscription.HasPremium)
				fmt.Printf("Entitlements: %v\n", w.Subscription.Entitlements)
				fmt.Printf("Business entities: %v\n", w.Capabilities.BusinessEntitiesAvailable)
			})
	},
}

func init() {
	RootCmd.AddCommand(whoamiCmd)
}
