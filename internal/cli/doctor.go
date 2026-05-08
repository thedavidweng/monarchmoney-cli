package cli

import (
	"fmt"
	"time"

	"github.com/monarchmoney-cli/monarch/internal/doctor"
	"github.com/monarchmoney-cli/monarch/internal/output"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check local configuration and connectivity",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		res := doctor.Check()

		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		
		if jsonMode {
			env := output.NewEnvelope("doctor", profile, "2026-05-08", "", res, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Println("Monarch Money CLI Doctor")
			fmt.Printf("Version: %s\n", res.Version)
			fmt.Printf("OS/Arch: %s/%s\n", res.OS, res.Arch)
			fmt.Printf("Config Path: %s (Exists: %v)\n", res.Config.Path, res.Config.Exists)
			fmt.Printf("Session Path: %s (Exists: %v)\n", res.Session.Path, res.Session.Exists)
		}
	},
}

func init() {
	RootCmd.AddCommand(doctorCmd)
}
