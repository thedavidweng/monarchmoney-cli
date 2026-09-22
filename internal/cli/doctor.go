package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/doctor"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

var connect bool

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Short:   "Check local configuration and connectivity",
	GroupID: "utility",
	Long:    `Verify version, config file, session existence and permissions without touching the network. Add --connect to also probe the Monarch API, which requires a logged-in session. Run this first when any command fails with auth or network errors.`,
	Example: "  monarch doctor\n  monarch doctor --connect --json",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		res := doctor.Check(cmd.Context(), connect)

		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if jsonMode {
			env := output.NewEnvelope("doctor", profile, output.SchemaVersion, requestID, res, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Println("Monarch Money CLI Doctor")
			fmt.Printf("Version: %s\n", res.Version)
			fmt.Printf("OS/Arch: %s/%s\n", res.OS, res.Arch)
			fmt.Printf("Config Path: %s (Exists: %v)\n", res.Config.Path, res.Config.Exists)
			fmt.Printf("Session Path: %s (Exists: %v, Auth: %v, PermOK: %v)\n", res.Session.Path, res.Session.Exists, res.Session.Authenticated, res.Session.PermissionOK)
			if connect {
				fmt.Printf("API Connected: %v\n", res.Network.APIReachable)
			}
		}
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&connect, "connect", false, "check API connectivity (requires auth)")
	RootCmd.AddCommand(doctorCmd)
}
