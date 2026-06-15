package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

var goalsCmd = &cobra.Command{
	Use:     "goals",
	Short:   "Manage Monarch Money goals",
	GroupID: "core",
	Example: "  monarch goals list --json",
}

var goalsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List goals",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "goals.list", start)
		if !ok {
			return
		}
		svc := deps.Service

		goals, err := svc.ListGoals(cmd.Context())
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list goals", errors.CatAPI, false, err)
			}
			handleError(renderer, "goals.list", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("goals.list", profile, output.SchemaVersion, "", goals, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("%-20s %s\n", "ID", "NAME")
			for _, goal := range goals {
				fmt.Printf("%-20s %s\n", goal.ID, goal.Name)
			}
		}
	},
}

func init() {
	goalsCmd.AddCommand(goalsListCmd)
	RootCmd.AddCommand(goalsCmd)
}
