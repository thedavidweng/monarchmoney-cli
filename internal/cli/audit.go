package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/audit"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

var auditCmd = &cobra.Command{
	Use:     "audit",
	Short:   "Manage audit logs",
	GroupID: "utility",
	Example: "  monarch audit cleanup --older-than 30",
}

var auditCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove audit log files older than N days",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if auditCleanupDays <= 0 {
			handleError(renderer, "audit.cleanup", errors.New(errors.InvalidArguments, "--older-than must be a positive number of days", errors.CatValidation, false, nil), start)
			return
		}

		logger := audit.NewLogger()
		removed, err := logger.Cleanup(auditCleanupDays)
		if err != nil {
			handleError(renderer, "audit.cleanup", errors.New(errors.InternalError, "failed to cleanup audit logs", errors.CatInternal, false, err), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("audit.cleanup", profile, output.SchemaVersion, "", map[string]interface{}{"removed": removed, "older_than_days": auditCleanupDays}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Removed %d audit log file(s) older than %d days.\n", removed, auditCleanupDays)
		}
	},
}

var auditCleanupDays int

func init() {
	auditCleanupCmd.Flags().IntVar(&auditCleanupDays, "older-than", 30, "remove logs older than N days (default 30)")

	auditCmd.AddCommand(auditCleanupCmd)
	RootCmd.AddCommand(auditCmd)
}
