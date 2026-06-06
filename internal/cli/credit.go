package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

var creditCmd = &cobra.Command{
	Use:     "credit",
	Short:   "Manage credit history",
	GroupID: "core",
	Example: "  monarch credit history --json",
}

var creditHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Get credit score history",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "credit.history", start)
		if !ok {
			return
		}
		svc := deps.Service

		history, err := svc.GetCreditHistory(cmd.Context())
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get credit history", errors.CatAPI, false, err)
			}
			handleError(renderer, "credit.history", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("credit.history", profile, output.SchemaVersion, "", history, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-12s %s\n", "DATE", "SCORE")
			for _, r := range history {
				fmt.Printf("%-12s %d\n", r.Date, r.Score)
			}
		}
	},
}

func init() {
	creditCmd.AddCommand(creditHistoryCmd)
	RootCmd.AddCommand(creditCmd)
}
