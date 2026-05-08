package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/monarchmoney-cli/monarch/internal/auth"
	"github.com/monarchmoney-cli/monarch/internal/config"
	"github.com/monarchmoney-cli/monarch/internal/errors"
	"github.com/monarchmoney-cli/monarch/internal/graphql"
	"github.com/monarchmoney-cli/monarch/internal/monarch"
	"github.com/monarchmoney-cli/monarch/internal/output"
	"github.com/spf13/cobra"
)

var (
	monthStr string
)

var budgetsCmd = &cobra.Command{
	Use:   "budgets",
	Short: "Manage Monarch Money budgets",
}

var budgetsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List budgets",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "budgets.list", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		opts := monarch.ListBudgetsOptions{}
		if monthStr != "" {
			parts := strings.Split(monthStr, "-")
			if len(parts) != 2 {
				handleError(renderer, "budgets.list", errors.New(errors.InvalidArguments, "invalid month format, use YYYY-MM", errors.CatValidation, false, nil), start)
				return
			}
			y, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])
			opts.Year = y
			opts.Month = m
		}

		budgets, err := svc.ListBudgets(cmd.Context(), opts)
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list budgets", errors.CatAPI, false, err)
			}
			handleError(renderer, "budgets.list", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("budgets.list", profile, "2026-05-08", "", budgets, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-30s %10s %10s %10s\n", "CATEGORY", "PLANNED", "ACTUAL", "REMAINING")
			for _, b := range budgets {
				fmt.Printf("%-30s %10.2f %10.2f %10.2f\n", b.CategoryName, b.Planned, b.Actual, b.Planned-b.Actual)
			}
		}
	},
}

func init() {
	budgetsListCmd.Flags().StringVar(&monthStr, "month", "", "month in YYYY-MM format")

	budgetsCmd.AddCommand(budgetsListCmd)
	RootCmd.AddCommand(budgetsCmd)
}
