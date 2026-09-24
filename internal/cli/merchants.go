package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
)

var (
	merchantSearch                     string
	merchantLimit                      int
	merchantOffset                     int
	merchantOrderBy                    string
	merchantHasDefaultCategory         bool
	merchantIncludeWithoutTransactions bool
	merchantIncludeIDs                 []string
	merchantName                       string
	merchantDefaultCategoryID          string
	merchantDefaultCategoryMode        string
	merchantShowDefaultCategoryPrompt  bool
	merchantRecurrenceFile             string
	merchantMoveTo                     string
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
				opts := &monarch.ListMerchantsOptions{
					Search:     merchantSearch,
					Limit:      merchantLimit,
					Offset:     merchantOffset,
					OrderBy:    merchantOrderBy,
					IncludeIDs: merchantIncludeIDs,
				}
				if cmd.Flags().Changed("has-default-category") {
					opts.HasDefaultCategory = &merchantHasDefaultCategory
				}
				if cmd.Flags().Changed("include-without-transactions") {
					opts.IncludeMerchantsWithoutTransactions = &merchantIncludeWithoutTransactions
				}
				return svc.ListMerchants(ctx, opts)
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
				if m.DefaultCategory != nil {
					fmt.Printf("Default category: %s (%s)\n", m.DefaultCategory.Name, m.DefaultCategory.ID)
				}
				if m.DefaultCategoryApplicationMode != "" {
					fmt.Printf("Default category mode: %s\n", m.DefaultCategoryApplicationMode)
				}
			})
	},
}

var merchantsUpdateCmd = &cobra.Command{
	Use:   "update <merchant-id>",
	Short: "Update a merchant (requires --confirm)",
	Long:  `Rename a merchant (--name), set its default category (--default-category-id with --default-category-mode new_only|new_and_edits), toggle the default-category prompt (--show-default-category-prompt), or replace recurrence settings from a JSON file (--recurrence-file). Pass at least one update flag.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "merchants.update", "failed to update merchant", safety.TierMutation, func() (mutation, *errors.Error) {
			input := &monarch.UpdateMerchantInput{ID: id, Name: merchantName}
			if merchantDefaultCategoryMode != "" && merchantDefaultCategoryMode != "new_only" && merchantDefaultCategoryMode != "new_and_edits" {
				return mutation{}, errors.New(errors.InvalidArguments, "--default-category-mode must be new_only or new_and_edits", errors.CatValidation, false, nil)
			}
			input.DefaultCategoryID = merchantDefaultCategoryID
			input.DefaultCategoryMode = merchantDefaultCategoryMode
			if cmd.Flags().Changed("show-default-category-prompt") {
				input.ShowDefaultCategoryPrompt = &merchantShowDefaultCategoryPrompt
			}
			if merchantRecurrenceFile != "" {
				data, err := os.ReadFile(merchantRecurrenceFile)
				if err != nil {
					return mutation{}, errors.New(errors.InvalidArguments, "failed to read recurrence file: "+err.Error(), errors.CatValidation, false, err)
				}
				var recurrence map[string]any
				if err := json.Unmarshal(data, &recurrence); err != nil {
					return mutation{}, errors.New(errors.InvalidArguments, "invalid recurrence JSON: "+err.Error(), errors.CatValidation, false, err)
				}
				if _, ok := recurrence["isRecurring"]; !ok {
					return mutation{}, errors.New(errors.InvalidArguments, "recurrence JSON must contain isRecurring", errors.CatValidation, false, nil)
				}
				input.Recurrence = recurrence
			}
			if input.Name == "" && input.DefaultCategoryID == "" && input.DefaultCategoryMode == "" && input.ShowDefaultCategoryPrompt == nil && input.Recurrence == nil {
				return mutation{}, errors.New(errors.InvalidArguments, "pass at least one of --name, --default-category-id, --default-category-mode, --show-default-category-prompt, --recurrence-file", errors.CatValidation, false, nil)
			}
			var merchant *monarch.Merchant
			return mutation{
				resourceID: id,
				planAfter:  map[string]any{"input": input},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateMerchant(ctx, input)
					if err != nil {
						return nil, err
					}
					merchant = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Successfully updated merchant %s to %q.\n", merchant.ID, merchant.Name) },
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
	merchantsListCmd.Flags().BoolVar(&merchantHasDefaultCategory, "has-default-category", false, "filter by whether a default category is set")
	merchantsListCmd.Flags().BoolVar(&merchantIncludeWithoutTransactions, "include-without-transactions", false, "include merchants without transactions")
	merchantsListCmd.Flags().StringSliceVar(&merchantIncludeIDs, "include-id", nil, "include these merchant IDs (repeatable)")

	merchantsUpdateCmd.Flags().StringVar(&merchantName, "name", "", "new merchant name")
	merchantsUpdateCmd.Flags().StringVar(&merchantDefaultCategoryID, "default-category-id", "", "default category ID for this merchant")
	merchantsUpdateCmd.Flags().StringVar(&merchantDefaultCategoryMode, "default-category-mode", "", "default category scope (new_only, new_and_edits)")
	merchantsUpdateCmd.Flags().BoolVar(&merchantShowDefaultCategoryPrompt, "show-default-category-prompt", false, "show the default-category prompt for this merchant")
	merchantsUpdateCmd.Flags().StringVar(&merchantRecurrenceFile, "recurrence-file", "", "JSON file with recurrence settings (must contain isRecurring)")

	merchantsDeleteCmd.Flags().StringVar(&merchantMoveTo, "move-to", "", "move relations to this merchant ID before deleting")

	merchantsCmd.AddCommand(merchantsListCmd)
	merchantsCmd.AddCommand(merchantsShowCmd)
	merchantsCmd.AddCommand(merchantsUpdateCmd)
	merchantsCmd.AddCommand(merchantsDeleteCmd)
	RootCmd.AddCommand(merchantsCmd)
}
