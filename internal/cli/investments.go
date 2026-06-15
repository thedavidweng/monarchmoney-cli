package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

var (
	investmentFrom          string
	investmentTo            string
	investmentAccountIDs    []string
	investmentSecurityIDs   []string
	investmentIncludeValues bool
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
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if err := validateOptionalDate("from", investmentFrom); err != nil {
			handleError(renderer, "investments.portfolio", err, start)
			return
		}
		if err := validateOptionalDate("to", investmentTo); err != nil {
			handleError(renderer, "investments.portfolio", err, start)
			return
		}

		deps, ok := newDeps(renderer, "investments.portfolio", start)
		if !ok {
			return
		}
		svc := deps.Service

		portfolio, err := svc.GetInvestmentPortfolio(cmd.Context(), monarch.InvestmentPortfolioOptions{
			StartDate:  investmentFrom,
			EndDate:    investmentTo,
			AccountIDs: investmentAccountIDs,
		})
		if err != nil {
			handleError(renderer, "investments.portfolio", wrapError(err, "failed to get investment portfolio"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("investments.portfolio", profile, output.SchemaVersion, "", portfolio, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("Total Value: %.2f\n", portfolio.Performance.TotalValue)
			fmt.Printf("%-20s %-10s %12s\n", "SECURITY", "TICKER", "VALUE")
			for _, holding := range portfolio.Holdings {
				fmt.Printf("%-20s %-10s %12.2f\n", holding.Security.Name, holding.Security.Ticker, holding.TotalValue)
			}
		}
	},
}

var investmentsPerformanceCmd = &cobra.Command{
	Use:   "performance",
	Short: "Get historical security performance",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if len(investmentSecurityIDs) == 0 {
			handleError(renderer, "investments.performance", errors.New(errors.InvalidArguments, "--security-id is required", errors.CatValidation, false, nil), start)
			return
		}
		if err := validateRequiredDate("from", investmentFrom); err != nil {
			handleError(renderer, "investments.performance", err, start)
			return
		}
		if err := validateRequiredDate("to", investmentTo); err != nil {
			handleError(renderer, "investments.performance", err, start)
			return
		}

		deps, ok := newDeps(renderer, "investments.performance", start)
		if !ok {
			return
		}
		svc := deps.Service

		performance, err := svc.GetSecurityPerformance(cmd.Context(), monarch.SecurityPerformanceOptions{
			SecurityIDs:   investmentSecurityIDs,
			StartDate:     investmentFrom,
			EndDate:       investmentTo,
			IncludeValues: investmentIncludeValues,
		})
		if err != nil {
			handleError(renderer, "investments.performance", wrapError(err, "failed to get security performance"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("investments.performance", profile, output.SchemaVersion, "", performance, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("%-20s %-10s %6s\n", "SECURITY", "TICKER", "POINTS")
			for _, item := range performance {
				fmt.Printf("%-20s %-10s %6d\n", item.Security.Name, item.Security.Ticker, len(item.HistoricalChart))
			}
		}
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
