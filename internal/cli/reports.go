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
	reportGroupBy   string
	reportTimeframe string
	reportSortBy    string
	reportSearch    string
	reportName      string
)

var reportsCmd = &cobra.Command{
	Use:     "reports",
	Short:   "Query report data and manage saved reports",
	GroupID: "analysis",
	Example: "  monarch reports data --from 2026-01-01 --to 2026-01-31 --group-by category --json",
}

var reportsDataCmd = &cobra.Command{
	Use:   "data",
	Short: "Query grouped transaction report data with totals",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "reports.data", "failed to get report data",
			func(ctx context.Context, svc *monarch.Service) (*monarch.ReportResult, error) {
				return svc.GetReportData(ctx, &monarch.ReportDataOptions{
					StartDate: txStartDate,
					EndDate:   txEndDate,
					Search:    reportSearch,
					GroupBy:   reportGroupBy,
					Timeframe: reportTimeframe,
					SortBy:    reportSortBy,
				})
			},
			func(result *monarch.ReportResult) {
				fmt.Printf("%-12s %-24s %12s %s\n", "DATE", "GROUP", "SUM", "COUNT")
				for _, row := range result.Rows {
					label := row.Category
					if label == "" {
						label = row.CategoryGroup
					}
					if label == "" {
						label = row.Merchant
					}
					sum := 0.0
					if row.Summary.Sum != nil {
						sum = *row.Summary.Sum
					}
					fmt.Printf("%-12s %-24s %12.2f %d\n", row.Date, label, sum, row.Summary.Count)
				}
			})
	},
}

var reportsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved reports",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "reports.list", "failed to list saved reports",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.SavedReport, error) {
				return svc.ListSavedReports(ctx)
			},
			func(reports []monarch.SavedReport) {
				fmt.Printf("%-20s %-30s %s\n", "ID", "NAME", "TIMEFRAME")
				for _, r := range reports {
					fmt.Printf("%-20s %-30s %s\n", r.ID, r.DisplayName, r.Timeframe)
				}
			})
	},
}

var reportsShowCmd = &cobra.Command{
	Use:   "show <report-id>",
	Short: "Show a saved report",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "reports.show", "failed to get saved report",
			func(ctx context.Context, svc *monarch.Service) (*monarch.SavedReport, error) {
				return svc.GetSavedReport(ctx, args[0])
			},
			func(r *monarch.SavedReport) {
				fmt.Printf("ID:        %s\n", r.ID)
				fmt.Printf("Name:      %s\n", r.DisplayName)
				fmt.Printf("Timeframe: %s\n", r.Timeframe)
			})
	},
}

var reportsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a saved report",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "reports.create", "failed to create saved report", safety.TierMutation, func() (mutation, *errors.Error) {
			var report *monarch.SavedReport
			return mutation{
				planAfter: map[string]string{"name": reportName, "group_by": reportGroupBy, "timeframe": reportTimeframe},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					created, err := svc.CreateSavedReport(ctx, reportName, reportGroupBy, reportTimeframe)
					if err != nil {
						return nil, err
					}
					report = created
					return created, nil
				},
				human: func() { fmt.Printf("Successfully created report %s (%s).\n", report.DisplayName, report.ID) },
			}, nil
		})
	},
}

var reportsUpdateCmd = &cobra.Command{
	Use:   "update <report-id>",
	Short: "Rename a saved report",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "reports.update", "failed to update saved report", safety.TierMutation, func() (mutation, *errors.Error) {
			var report *monarch.SavedReport
			return mutation{
				resourceID: id,
				planAfter:  map[string]string{"name": reportName},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateSavedReport(ctx, id, reportName)
					if err != nil {
						return nil, err
					}
					report = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Successfully renamed report %s to %q.\n", report.ID, report.DisplayName) },
			}, nil
		})
	},
}

var reportsDeleteCmd = &cobra.Command{
	Use:   "delete <report-id>",
	Short: "Delete a saved report",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "reports.delete", "failed to delete saved report", safety.TierDestructive, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.DeleteSavedReport(ctx, id); err != nil {
						return nil, err
					}
					return map[string]string{"status": "deleted"}, nil
				},
				human: func() { fmt.Printf("Successfully deleted report %s.\n", id) },
			}, nil
		})
	},
}

func init() {
	reportsDataCmd.Flags().StringVar(&reportGroupBy, "group-by", "category", "group rows by (category, category-group, merchant, none)")
	reportsDataCmd.Flags().StringVar(&reportTimeframe, "timeframe", "", "time bucket (day, week, month, quarter, year)")
	reportsDataCmd.Flags().StringVar(&reportSortBy, "sort-by", "", "sort rows by (sum, sum_income, sum_expense, count, avg, max)")
	reportsDataCmd.Flags().StringVar(&reportSearch, "search", "", "filter transactions by search text")

	reportsCreateCmd.Flags().StringVar(&reportName, "name", "", "report display name")
	reportsCreateCmd.Flags().StringVar(&reportGroupBy, "group-by", "category", "default dimension")
	reportsCreateCmd.Flags().StringVar(&reportTimeframe, "timeframe", "", "default timeframe")
	reportsCreateCmd.MarkFlagRequired("name") //nolint:errcheck // flag registered above

	reportsUpdateCmd.Flags().StringVar(&reportName, "name", "", "new report display name")
	reportsUpdateCmd.MarkFlagRequired("name") //nolint:errcheck // flag registered above

	reportsCmd.AddCommand(reportsDataCmd)
	reportsCmd.AddCommand(reportsListCmd)
	reportsCmd.AddCommand(reportsShowCmd)
	reportsCmd.AddCommand(reportsCreateCmd)
	reportsCmd.AddCommand(reportsUpdateCmd)
	reportsCmd.AddCommand(reportsDeleteCmd)
	RootCmd.AddCommand(reportsCmd)
}
