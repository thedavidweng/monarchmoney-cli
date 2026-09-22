package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
)

var (
	merchantSearch  string
	merchantLimit   int
	merchantOffset  int
	merchantOrderBy string
	merchantName    string
	merchantMoveTo  string
)

var merchantsCmd = &cobra.Command{
	Use:     "merchants",
	Short:   "Manage Monarch Money merchants",
	GroupID: "core",
	Example: "  monarch merchants list --search \"Whole\" --json\n  monarch merchants update <id> --name \"Whole Foods\" --confirm",
}

var merchantsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List merchants",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "merchants.list", "failed to list merchants",
			func(ctx context.Context, svc *monarch.Service) ([]*monarch.Merchant, error) {
				return svc.ListMerchants(ctx, merchantSearch, merchantLimit, merchantOffset, merchantOrderBy)
			},
			func(merchants []*monarch.Merchant) {
				fmt.Printf("%-20s %-30s %s\n", "ID", "NAME", "TRANSACTIONS")
				for _, m := range merchants {
					fmt.Printf("%-20s %-30s %d\n", m.ID, m.Name, m.TransactionCount)
				}
			})
	},
}

var merchantsShowCmd = &cobra.Command{
	Use:   "show <merchant-id>",
	Short: "Show merchant details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "merchants.show", "failed to get merchant",
			func(ctx context.Context, svc *monarch.Service) (*monarch.Merchant, error) {
				return svc.GetMerchant(ctx, args[0])
			},
			func(m *monarch.Merchant) {
				fmt.Printf("ID:           %s\n", m.ID)
				fmt.Printf("Name:         %s\n", m.Name)
				fmt.Printf("Transactions: %d\n", m.TransactionCount)
				fmt.Printf("Rules:        %d\n", m.RuleCount)
			})
	},
}

var merchantsUpdateCmd = &cobra.Command{
	Use:   "update <merchant-id>",
	Short: "Rename a merchant (requires --confirm)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "merchants.update", "failed to update merchant", safety.TierMutation, func() (mutation, *errors.Error) {
			var merchant *monarch.Merchant
			return mutation{
				resourceID: id,
				planAfter:  map[string]string{"name": merchantName},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateMerchant(ctx, id, merchantName)
					if err != nil {
						return nil, err
					}
					merchant = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Successfully renamed merchant %s to %q.\n", merchant.ID, merchant.Name) },
			}, nil
		})
	},
}

var merchantsDeleteCmd = &cobra.Command{
	Use:   "delete <merchant-id>",
	Short: "Delete a merchant, optionally moving relations elsewhere (requires --confirm)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "merchants.delete", "failed to delete merchant", safety.TierDestructive, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				planAfter:  map[string]string{"move_to": merchantMoveTo},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.DeleteMerchant(ctx, id, merchantMoveTo); err != nil {
						return nil, err
					}
					return map[string]string{"status": "deleted"}, nil
				},
				human: func() { fmt.Printf("Successfully deleted merchant %s.\n", id) },
			}, nil
		})
	},
}

func init() {
	merchantsListCmd.Flags().StringVar(&merchantSearch, "search", "", "filter merchants by name")
	merchantsListCmd.Flags().IntVar(&merchantLimit, "limit", 0, "maximum number of merchants to return")
	merchantsListCmd.Flags().IntVar(&merchantOffset, "offset", 0, "number of merchants to skip")
	merchantsListCmd.Flags().StringVar(&merchantOrderBy, "order-by", "", "sort order (e.g. transaction_count, name)")

	merchantsUpdateCmd.Flags().StringVar(&merchantName, "name", "", "new merchant name")
	merchantsUpdateCmd.MarkFlagRequired("name") //nolint:errcheck // flag registered above

	merchantsDeleteCmd.Flags().StringVar(&merchantMoveTo, "move-to", "", "move relations to this merchant ID before deleting")

	merchantsCmd.AddCommand(merchantsListCmd)
	merchantsCmd.AddCommand(merchantsShowCmd)
	merchantsCmd.AddCommand(merchantsUpdateCmd)
	merchantsCmd.AddCommand(merchantsDeleteCmd)
	RootCmd.AddCommand(merchantsCmd)
}
