package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
)

var (
	investmentFrom          string
	investmentTo            string
	investmentAccountIDs    []string
	investmentSecurityIDs   []string
	investmentIncludeValues bool
	investmentLimit         int
	investmentAccountID     string
	investmentSecurityID    string
	investmentQuantity      float64
	investmentCostBasis     float64
	investmentSecurityType  string
)

var investmentsCmd = &cobra.Command{
	Use:     "investments",
	Short:   "Inspect Monarch Money investments",
	GroupID: "core",
	Example: "  monarch investments portfolio --json\n  monarch investments performance --json",
}

var investmentsPortfolioCmd = &cobra.Command{
	Use:   "portfolio",
	Short: "Get investment portfolio holdings and performance",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "investments.portfolio", "failed to get investment portfolio",
			func(ctx context.Context, svc *monarch.Service) (*monarch.InvestmentPortfolio, error) {
				if err := validateOptionalDate("from", investmentFrom); err != nil {
					return nil, err
				}
				if err := validateOptionalDate("to", investmentTo); err != nil {
					return nil, err
				}
				return svc.GetInvestmentPortfolio(ctx, monarch.InvestmentPortfolioOptions{
					StartDate:  investmentFrom,
					EndDate:    investmentTo,
					AccountIDs: investmentAccountIDs,
				})
			},
			func(portfolio *monarch.InvestmentPortfolio) {
				fmt.Printf("Total Value: %.2f\n", portfolio.Performance.TotalValue)
				fmt.Printf("%-20s %-10s %12s\n", "SECURITY", "TICKER", "VALUE")
				for _, holding := range portfolio.Holdings {
					fmt.Printf("%-20s %-10s %12.2f\n", holding.Security.Name, holding.Security.Ticker, holding.TotalValue)
				}
			})
	},
}

var investmentsPerformanceCmd = &cobra.Command{
	Use:   "performance",
	Short: "Get historical security performance",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "investments.performance", "failed to get security performance",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.SecurityPerformance, error) {
				if len(investmentSecurityIDs) == 0 {
					return nil, errors.New(errors.InvalidArguments, "--security-id is required", errors.CatValidation, false, nil)
				}
				if err := validateRequiredDate("from", investmentFrom); err != nil {
					return nil, err
				}
				if err := validateRequiredDate("to", investmentTo); err != nil {
					return nil, err
				}
				return svc.GetSecurityPerformance(ctx, monarch.SecurityPerformanceOptions{
					SecurityIDs:   investmentSecurityIDs,
					StartDate:     investmentFrom,
					EndDate:       investmentTo,
					IncludeValues: investmentIncludeValues,
				})
			},
			func(performance []monarch.SecurityPerformance) {
				fmt.Printf("%-20s %-10s %6s\n", "SECURITY", "TICKER", "POINTS")
				for _, item := range performance {
					fmt.Printf("%-20s %-10s %6d\n", item.Security.Name, item.Security.Ticker, len(item.HistoricalChart))
				}
			})
	},
}

var investmentsAccountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "List investment (brokerage) accounts",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "investments.accounts", "failed to list investment accounts",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.InvestmentAccount, error) {
				return svc.ListInvestmentAccounts(ctx)
			},
			func(accounts []monarch.InvestmentAccount) {
				fmt.Printf("%-20s %s\n", "ID", "NAME")
				for _, a := range accounts {
					fmt.Printf("%-20s %s\n", a.ID, a.DisplayName)
				}
			})
	},
}

var investmentsHoldingsCmd = &cobra.Command{
	Use:   "holdings",
	Short: "Manage holdings",
}

var investmentsHoldingsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List holdings, optionally filtered by account",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "investments.holdings.list", "failed to list holdings",
			func(ctx context.Context, svc *monarch.Service) ([]*monarch.HoldingDetail, error) {
				return svc.ListHoldingDetails(ctx, investmentAccountIDs)
			},
			func(holdings []*monarch.HoldingDetail) {
				fmt.Printf("%-20s %-10s %12s %12s %s\n", "ID", "TICKER", "QUANTITY", "VALUE", "ACCOUNT")
				for _, h := range holdings {
					fmt.Printf("%-20s %-10s %12.4f %12.2f %s\n", h.ID, h.Ticker, h.Quantity, h.TotalValue, h.AccountName)
				}
			})
	},
}

var investmentsHoldingShowCmd = &cobra.Command{
	Use:   "show <holding-id>",
	Short: "Show a holding",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "investments.holding.show", "failed to get holding",
			func(ctx context.Context, svc *monarch.Service) (*monarch.HoldingDetail, error) {
				return svc.GetHoldingDetail(ctx, args[0])
			},
			func(h *monarch.HoldingDetail) {
				fmt.Printf("ID:       %s\n", h.ID)
				fmt.Printf("Ticker:   %s\n", h.Ticker)
				fmt.Printf("Quantity: %.4f\n", h.Quantity)
				fmt.Printf("Value:    %.2f\n", h.TotalValue)
				fmt.Printf("Account:  %s\n", h.AccountName)
			})
	},
}

var investmentsHoldingsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a manual holding (requires --confirm)",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "investments.holdings.create", "failed to create manual holding", safety.TierMutation, func() (mutation, *errors.Error) {
			if investmentAccountID == "" || investmentSecurityID == "" {
				return mutation{}, errors.New(errors.InvalidArguments, "--account and --security are required", errors.CatValidation, false, nil)
			}
			input := &monarch.ManualHoldingInput{
				AccountID:  investmentAccountID,
				SecurityID: investmentSecurityID,
				Quantity:   investmentQuantity,
			}
			after := map[string]any{"account": investmentAccountID, "security": investmentSecurityID, "quantity": investmentQuantity}
			if cmd.Flags().Changed("cost-basis") {
				input.CostBasis = &investmentCostBasis
				after["cost_basis"] = investmentCostBasis
			}
			var holding *monarch.HoldingDetail
			return mutation{
				planAfter: after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					created, err := svc.CreateManualHolding(ctx, input)
					if err != nil {
						return nil, err
					}
					holding = created
					return created, nil
				},
				human: func() { fmt.Printf("Successfully created holding %s.\n", holding.ID) },
			}, nil
		})
	},
}

var investmentsHoldingsUpdateCmd = &cobra.Command{
	Use:   "update <holding-id>",
	Short: "Update a manual holding (requires --confirm)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "investments.holdings.update", "failed to update holding", safety.TierMutation, func() (mutation, *errors.Error) {
			update := &monarch.ManualHoldingUpdate{}
			after := map[string]any{}
			if cmd.Flags().Changed("quantity") {
				update.Quantity = &investmentQuantity
				after["quantity"] = investmentQuantity
			}
			if cmd.Flags().Changed("cost-basis") {
				update.CostBasis = &investmentCostBasis
				after["cost_basis"] = investmentCostBasis
			}
			if cmd.Flags().Changed("type") {
				update.SecurityType = &investmentSecurityType
				after["type"] = investmentSecurityType
			}
			if len(after) == 0 {
				return mutation{}, errors.New(errors.InvalidArguments, "at least one of --quantity, --cost-basis, --type is required", errors.CatValidation, false, nil)
			}
			var holding *monarch.HoldingDetail
			return mutation{
				resourceID: id,
				planAfter:  after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateManualHolding(ctx, id, update)
					if err != nil {
						return nil, err
					}
					holding = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Successfully updated holding %s.\n", holding.ID) },
			}, nil
		})
	},
}

var investmentsHoldingsDeleteCmd = &cobra.Command{
	Use:   "delete <holding-id>",
	Short: "Delete a manual holding (requires --confirm)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "investments.holdings.delete", "failed to delete holding", safety.TierDestructive, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.DeleteManualHolding(ctx, id); err != nil {
						return nil, err
					}
					return map[string]string{"status": "deleted"}, nil
				},
				human: func() { fmt.Printf("Successfully deleted holding %s.\n", id) },
			}, nil
		})
	},
}

var investmentsSecuritiesCmd = &cobra.Command{
	Use:   "securities <query>",
	Short: "Search securities by name or ticker",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "investments.securities", "failed to search securities",
			func(ctx context.Context, svc *monarch.Service) ([]*monarch.Security, error) {
				return svc.SearchSecurities(ctx, args[0], investmentLimit)
			},
			func(securities []*monarch.Security) {
				fmt.Printf("%-20s %-10s %-30s %s\n", "ID", "TICKER", "NAME", "PRICE")
				for _, s := range securities {
					fmt.Printf("%-20s %-10s %-30s %.2f\n", s.ID, s.Ticker, s.Name, s.CurrentPrice)
				}
			})
	},
}

var investmentsSecurityCmd = &cobra.Command{
	Use:   "security <security-id>",
	Short: "Show a security",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "investments.security", "failed to get security",
			func(ctx context.Context, svc *monarch.Service) (*monarch.Security, error) {
				return svc.GetSecurity(ctx, args[0])
			},
			func(s *monarch.Security) {
				fmt.Printf("ID:     %s\n", s.ID)
				fmt.Printf("Name:   %s\n", s.Name)
				fmt.Printf("Ticker: %s\n", s.Ticker)
				fmt.Printf("Price:  %.2f\n", s.CurrentPrice)
			})
	},
}

func init() {
	investmentsPortfolioCmd.Flags().StringVar(&investmentFrom, "from", "", "start date (YYYY-MM-DD)")
	investmentsPortfolioCmd.Flags().StringVar(&investmentTo, "to", "", "end date (YYYY-MM-DD)")
	investmentsPortfolioCmd.Flags().StringSliceVar(&investmentAccountIDs, "account-id", nil, "account id filter (repeatable)")

	investmentsPerformanceCmd.Flags().StringSliceVar(&investmentSecurityIDs, "security-id", nil, "security id to include (repeatable)")
	investmentsPerformanceCmd.Flags().StringVar(&investmentFrom, "from", "", "start date (YYYY-MM-DD)")
	investmentsPerformanceCmd.Flags().StringVar(&investmentTo, "to", "", "end date (YYYY-MM-DD)")
	investmentsPerformanceCmd.Flags().BoolVar(&investmentIncludeValues, "values", false, "include chart value fields")

	investmentsHoldingsListCmd.Flags().StringSliceVar(&investmentAccountIDs, "account-id", nil, "account id filter (repeatable)")

	investmentsSecuritiesCmd.Flags().IntVar(&investmentLimit, "limit", 20, "maximum number of securities to return")

	investmentsHoldingsCreateCmd.Flags().StringVar(&investmentAccountID, "account", "", "account ID")
	investmentsHoldingsCreateCmd.Flags().StringVar(&investmentSecurityID, "security", "", "security ID")
	investmentsHoldingsCreateCmd.Flags().Float64Var(&investmentQuantity, "quantity", 0, "quantity")
	investmentsHoldingsCreateCmd.Flags().Float64Var(&investmentCostBasis, "cost-basis", 0, "cost basis")
	investmentsHoldingsCreateCmd.MarkFlagRequired("account")  //nolint:errcheck // flag registered above
	investmentsHoldingsCreateCmd.MarkFlagRequired("security") //nolint:errcheck // flag registered above
	investmentsHoldingsCreateCmd.MarkFlagRequired("quantity") //nolint:errcheck // flag registered above

	investmentsHoldingsUpdateCmd.Flags().Float64Var(&investmentQuantity, "quantity", 0, "new quantity")
	investmentsHoldingsUpdateCmd.Flags().Float64Var(&investmentCostBasis, "cost-basis", 0, "new cost basis")
	investmentsHoldingsUpdateCmd.Flags().StringVar(&investmentSecurityType, "type", "", "new security type")

	investmentsHoldingsCmd.AddCommand(investmentsHoldingsListCmd)
	investmentsHoldingsCmd.AddCommand(investmentsHoldingShowCmd)
	investmentsHoldingsCmd.AddCommand(investmentsHoldingsCreateCmd)
	investmentsHoldingsCmd.AddCommand(investmentsHoldingsUpdateCmd)
	investmentsHoldingsCmd.AddCommand(investmentsHoldingsDeleteCmd)
	investmentsCmd.AddCommand(investmentsAccountsCmd)
	investmentsCmd.AddCommand(investmentsHoldingsCmd)
	investmentsCmd.AddCommand(investmentsSecuritiesCmd)
	investmentsCmd.AddCommand(investmentsSecurityCmd)
	investmentsCmd.AddCommand(investmentsPortfolioCmd)
	investmentsCmd.AddCommand(investmentsPerformanceCmd)
	RootCmd.AddCommand(investmentsCmd)
}

func validateOptionalDate(name, value string) *errors.Error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return errors.New(errors.InvalidArguments, name+" date must use YYYY-MM-DD", errors.CatValidation, false, err)
	}
	return nil
}

func validateRequiredDate(name, value string) *errors.Error {
	if value == "" {
		return errors.New(errors.InvalidArguments, "--"+name+" is required", errors.CatValidation, false, nil)
	}
	return validateOptionalDate(name, value)
}
