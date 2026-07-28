package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/config"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
	"github.com/thedavidweng/monarchmoney-cli/internal/version"
)

var (
	cfgFile   string
	jsonMode  bool
	pretty    bool
	events    bool
	readOnly  bool
	dryRun    bool
	confirm   bool
	timeout   time.Duration
	profile   string
	verbose   bool
	requestID string
)

var RootCmd = &cobra.Command{
	Use:     "monarch",
	Short:   "A local, agent-friendly CLI for Monarch Money",
	Version: version.GetVersion(),
	Long: `monarchmoney-cli is a single-binary command line tool for working with
Monarch Money data from your terminal, scripts, and local agents.`,
	Example: `  monarch accounts list --json
  monarch transactions search "Amazon" --from 2024-01-01
  monarch transactions update tx_123 --category cat_food --dry-run
  monarch cashflow spending --from 2024-01-01 --to 2024-01-31
  monarch rules list --json`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		requestID = uuid.NewString()

		if cfgFile == "" {
			cfgFile = os.Getenv("MONARCH_CONFIG")
		}

		cfg, _ := config.Load(cfgFile)

		jsonMode = jsonMode || envBool("MONARCH_JSON")
		pretty = pretty || envBool("MONARCH_PRETTY")
		events = events || envBool("MONARCH_EVENTS")
		readOnly = readOnly || envBool("MONARCH_READONLY") || envBool("MONARCH_READ_ONLY")
		dryRun = dryRun || envBool("MONARCH_DRY_RUN")
		confirm = confirm || envBool("MONARCH_CONFIRM")
		verbose = verbose || envBool("MONARCH_VERBOSE")

		if !persistentFlagChanged(cmd, "profile") {
			profile = cfg.Profile
		}
		if !persistentFlagChanged(cmd, "timeout") {
			timeout = cfg.Timeout
		}
	},
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func envBool(key string) bool {
	return config.ParseBool(os.Getenv(key))
}

func persistentFlagChanged(cmd *cobra.Command, name string) bool {
	f := cmd.Root().PersistentFlags().Lookup(name)
	return f != nil && f.Changed
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		if e, ok := err.(*errors.Error); ok {
			fmt.Println(err)
			os.Exit(e.ExitCode())
		}
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.AddGroup(&cobra.Group{ID: "core", Title: "Core Commands"})
	RootCmd.AddGroup(&cobra.Group{ID: "analysis", Title: "Analysis & Insights"})
	RootCmd.AddGroup(&cobra.Group{ID: "utility", Title: "Utilities"})

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.monarchmoney-cli/config.yaml)")
	RootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "emit machine-readable JSON")
	RootCmd.PersistentFlags().BoolVar(&pretty, "pretty", false, "pretty-print JSON output")
	RootCmd.PersistentFlags().BoolVar(&events, "events", false, "emit NDJSON progress events (accounts refresh --wait)")
	RootCmd.PersistentFlags().BoolVar(&readOnly, "read-only", false, "block remote writes")
	RootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "preview a remote write without executing it")
	RootCmd.PersistentFlags().BoolVar(&confirm, "confirm", false, "explicitly execute a remote write")
	RootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 30*time.Second, "set command timeout")
	RootCmd.PersistentFlags().StringVar(&profile, "profile", "default", "use a named profile")
	RootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "print more diagnostics to stderr")

	RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of monarch",
	Run: func(cmd *cobra.Command, args []string) {
		if err := writeVersion(cmd.OutOrStdout(), profile, jsonMode, pretty, time.Duration(0)); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), err)
			os.Exit(1)
		}
	},
}

type versionPayload struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	BuiltBy string `json:"built_by"`
}

func writeVersion(out io.Writer, profileName string, jsonOut, prettyOut bool, duration time.Duration) error {
	if jsonOut {
		renderer := output.NewRenderer(out, nil, true, prettyOut)
		env := output.NewEnvelope("version", profileName, output.SchemaVersion, requestID, versionPayload{
			Version: version.GetVersion(),
			Commit:  version.GetCommit(),
			Date:    version.GetDate(),
			BuiltBy: version.GetBuiltBy(),
		}, duration)
		renderer.RenderSuccess(env)
		return nil
	}

	fmt.Fprint(out, monarchBanner)
	fmt.Fprintln(out)
	_, err := fmt.Fprintf(out, "monarch version %s (commit: %s, date: %s, built by: %s)\n", version.GetVersion(), version.GetCommit(), version.GetDate(), version.GetBuiltBy())
	return err
}
