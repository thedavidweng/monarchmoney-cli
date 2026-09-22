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
	householdDisplayName         string
	householdTimezone            string
	householdPrefNewNeedReview   bool
	householdPrefUncatNeedReview bool
	householdPrefPendingEditable bool
	householdPrefHiddenBeta      bool
	householdPrefExcludeBusiness bool
)

var householdCmd = &cobra.Command{
	Use:     "household",
	Short:   "Manage household, members, and preferences",
	GroupID: "core",
	Example: "  monarch household show --json\n  monarch household members --json",
}

var householdShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the current household",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "household.show", "failed to get household",
			func(ctx context.Context, svc *monarch.Service) (*monarch.Household, error) {
				return svc.GetHousehold(ctx)
			},
			func(h *monarch.Household) {
				fmt.Printf("ID:      %s\n", h.ID)
				fmt.Printf("Name:    %s\n", h.Name)
				fmt.Printf("Address: %s, %s %s %s %s\n", h.Address, h.City, h.State, h.ZipCode, h.Country)
			})
	},
}

var householdMembersCmd = &cobra.Command{
	Use:   "members",
	Short: "List household members",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "household.members", "failed to list household members",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.HouseholdMember, error) {
				return svc.ListHouseholdMembers(ctx)
			},
			func(members []monarch.HouseholdMember) {
				fmt.Printf("%-20s %-24s %-32s %s\n", "ID", "NAME", "EMAIL", "ROLE")
				for _, m := range members {
					fmt.Printf("%-20s %-24s %-32s %s\n", m.ID, m.DisplayName, m.Email, m.Role)
				}
			})
	},
}

var householdMemberCmd = &cobra.Command{
	Use:   "member <member-id>",
	Short: "Show a household member",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "household.member", "failed to get household member",
			func(ctx context.Context, svc *monarch.Service) (*monarch.HouseholdMember, error) {
				return svc.GetHouseholdMember(ctx, args[0])
			},
			func(m *monarch.HouseholdMember) {
				fmt.Printf("ID:    %s\n", m.ID)
				fmt.Printf("Name:  %s\n", m.DisplayName)
				fmt.Printf("Email: %s\n", m.Email)
				fmt.Printf("Role:  %s\n", m.Role)
			})
	},
}

var householdMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the current user profile",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "household.me", "failed to get current user",
			func(ctx context.Context, svc *monarch.Service) (*monarch.UserProfile, error) {
				return svc.GetCurrentUser(ctx)
			},
			func(u *monarch.UserProfile) {
				fmt.Printf("ID:       %s\n", u.ID)
				fmt.Printf("Email:    %s\n", u.Email)
				fmt.Printf("Name:     %s\n", u.DisplayName)
				fmt.Printf("Timezone: %s\n", u.Timezone)
			})
	},
}

var householdMeUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the current user profile (requires --confirm)",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "household.me.update", "failed to update current user", safety.TierMutation, func() (mutation, *errors.Error) {
			var displayName, timezone *string
			after := map[string]any{}
			if cmd.Flags().Changed("display-name") {
				displayName = &householdDisplayName
				after["display_name"] = householdDisplayName
			}
			if cmd.Flags().Changed("timezone") {
				timezone = &householdTimezone
				after["timezone"] = householdTimezone
			}
			if len(after) == 0 {
				return mutation{}, errors.New(errors.InvalidArguments, "at least one of --display-name, --timezone is required", errors.CatValidation, false, nil)
			}
			var user *monarch.UserProfile
			return mutation{
				planAfter: after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateCurrentUser(ctx, displayName, timezone)
					if err != nil {
						return nil, err
					}
					user = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Successfully updated user %s.\n", user.ID) },
			}, nil
		})
	},
}

var householdPreferencesCmd = &cobra.Command{
	Use:   "preferences",
	Short: "Show household preferences",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "household.preferences", "failed to get household preferences",
			func(ctx context.Context, svc *monarch.Service) (*monarch.HouseholdPreferences, error) {
				return svc.GetHouseholdPreferences(ctx)
			},
			func(p *monarch.HouseholdPreferences) {
				fmt.Printf("Budget system: %s\n", p.BudgetSystem)
				fmt.Printf("New transactions need review: %v\n", boolValue(p.NewTransactionsNeedReview))
				fmt.Printf("Uncategorized need review:    %v\n", boolValue(p.UncategorizedNeedReview))
				fmt.Printf("Pending can be edited:        %v\n", boolValue(p.PendingCanBeEdited))
			})
	},
}

var householdPreferencesUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update household review preferences (requires --confirm)",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "household.preferences.update", "failed to update household preferences", safety.TierMutation, func() (mutation, *errors.Error) {
			update := &monarch.HouseholdPreferencesUpdate{}
			after := map[string]any{}
			if cmd.Flags().Changed("new-need-review") {
				update.NewTransactionsNeedReview = &householdPrefNewNeedReview
				after["new_need_review"] = householdPrefNewNeedReview
			}
			if cmd.Flags().Changed("uncategorized-need-review") {
				update.UncategorizedNeedReview = &householdPrefUncatNeedReview
				after["uncategorized_need_review"] = householdPrefUncatNeedReview
			}
			if cmd.Flags().Changed("pending-editable") {
				update.PendingCanBeEdited = &householdPrefPendingEditable
				after["pending_editable"] = householdPrefPendingEditable
			}
			if cmd.Flags().Changed("hidden-beta") {
				update.HiddenTransactionsBetaEnabled = &householdPrefHiddenBeta
				after["hidden_beta"] = householdPrefHiddenBeta
			}
			if cmd.Flags().Changed("exclude-business") {
				update.ExcludeBusinessFromBudget = &householdPrefExcludeBusiness
				after["exclude_business"] = householdPrefExcludeBusiness
			}
			if len(after) == 0 {
				return mutation{}, errors.New(errors.InvalidArguments, "at least one preference flag is required", errors.CatValidation, false, nil)
			}
			var prefs *monarch.HouseholdPreferences
			return mutation{
				planAfter: after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateHouseholdPreferences(ctx, update)
					if err != nil {
						return nil, err
					}
					prefs = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Household preferences updated (budget system: %s).\n", prefs.BudgetSystem) },
			}, nil
		})
	},
}

func init() {
	householdMeUpdateCmd.Flags().StringVar(&householdDisplayName, "display-name", "", "display name")
	householdMeUpdateCmd.Flags().StringVar(&householdTimezone, "timezone", "", "timezone")

	householdPreferencesUpdateCmd.Flags().BoolVar(&householdPrefNewNeedReview, "new-need-review", false, "new transactions need review")
	householdPreferencesUpdateCmd.Flags().BoolVar(&householdPrefUncatNeedReview, "uncategorized-need-review", false, "uncategorized transactions need review")
	householdPreferencesUpdateCmd.Flags().BoolVar(&householdPrefPendingEditable, "pending-editable", false, "pending transactions can be edited")
	householdPreferencesUpdateCmd.Flags().BoolVar(&householdPrefHiddenBeta, "hidden-beta", false, "enable hidden transactions beta")
	householdPreferencesUpdateCmd.Flags().BoolVar(&householdPrefExcludeBusiness, "exclude-business", false, "exclude business from budget")

	householdMeCmd.AddCommand(householdMeUpdateCmd)
	householdPreferencesCmd.AddCommand(householdPreferencesUpdateCmd)
	householdCmd.AddCommand(householdShowCmd)
	householdCmd.AddCommand(householdMembersCmd)
	householdCmd.AddCommand(householdMemberCmd)
	householdCmd.AddCommand(householdMeCmd)
	householdCmd.AddCommand(householdPreferencesCmd)
	RootCmd.AddCommand(householdCmd)
}
