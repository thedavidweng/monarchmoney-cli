package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

var institutionsCmd = &cobra.Command{
	Use:     "institutions",
	Short:   "Manage financial institutions",
	GroupID: "core",
	Example: "  monarch institutions list --json",
}

var institutionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all institutions",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "institutions.list", start)
		if !ok {
			return
		}
		svc := deps.Service

		insts, err := svc.ListInstitutions(cmd.Context())
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list institutions", errors.CatAPI, false, err)
			}
			handleError(renderer, "institutions.list", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("institutions.list", profile, output.SchemaVersion, "", insts, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-20s %-30s %s\n", "ID", "NAME", "URL")
			for _, inst := range insts {
				fmt.Printf("%-20s %-30s %s\n", inst.ID, inst.Name, inst.URL)
			}
		}
	},
}

func init() {
	institutionsCmd.AddCommand(institutionsListCmd)
	RootCmd.AddCommand(institutionsCmd)
}
