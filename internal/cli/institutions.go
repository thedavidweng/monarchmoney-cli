package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
)

var institutionsCmd = &cobra.Command{
	Use:     "institutions",
	Short:   "Manage financial institutions",
	GroupID: "core",
	Example: "  monarch institutions list --json",
}

var institutionsStaleDays int

var institutionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all institutions",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "institutions.list", "failed to list institutions",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.Institution, error) {
				return svc.ListInstitutions(ctx)
			},
			func(insts []monarch.Institution) {
				fmt.Printf("%-20s %-30s %s\n", "ID", "NAME", "URL")
				for _, inst := range insts {
					fmt.Printf("%-20s %-30s %s\n", inst.ID, inst.Name, inst.URL)
				}
			})
	},
}

var institutionsHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Report the health of each linked institution connection",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "institutions.health", "failed to get connection health",
			func(ctx context.Context, svc *monarch.Service) (*monarch.SyncHealthReport, error) {
				return svc.GetConnectionHealth(ctx, institutionsStaleDays)
			},
			func(report *monarch.SyncHealthReport) {
				fmt.Printf("Connections: %d (%d need attention)\n", report.ConnectionCount, report.NeedsAttentionCount)
				for _, c := range report.Connections {
					status := "ok"
					if c.NeedsAttention {
						status = "needs attention"
					}
					fmt.Printf("%-30s %-16s %s\n", c.Institution, status, c.LastUpdated)
					for _, r := range c.Reasons {
						fmt.Printf("  - %s\n", r)
					}
				}
			})
	},
}

func init() {
	institutionsHealthCmd.Flags().IntVar(&institutionsStaleDays, "stale-after-days", 3, "flag connections with no successful update for this many days")
	institutionsCmd.AddCommand(institutionsListCmd)
	institutionsCmd.AddCommand(institutionsHealthCmd)
	RootCmd.AddCommand(institutionsCmd)
}
