package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thedavidweng/monarchmoney-cli/internal/config"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
	"github.com/thedavidweng/monarchmoney-cli/internal/version"
)

var (
	cfgFile  string
	jsonMode bool
	pretty   bool
	events   bool
	readOnly bool
	dryRun   bool
	confirm  bool
	timeout  time.Duration
	profile  string
	verbose  bool
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "monarch",
	Short:   "A local, agent-friendly CLI for Monarch Money",
	Version: version.Version,
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
		// Update variables from viper (which handles env vars)
		jsonMode = viper.GetBool("json")
		pretty = viper.GetBool("pretty")
		events = viper.GetBool("events")
		readOnly = viper.GetBool("read-only")
		dryRun = viper.GetBool("dry-run")
		confirm = viper.GetBool("confirm")
		timeout = viper.GetDuration("timeout")
		profile = viper.GetString("profile")
		verbose = viper.GetBool("verbose")
	},
}

// must panics if err is non-nil. Use for programmer errors that should never
// occur at runtime (e.g., registering a completion function for a known flag).
func must(err error) {
	if err != nil {
		panic(err)
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
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
	cobra.OnInitialize(initConfig)

	// Command groups for organized help output
	RootCmd.AddGroup(&cobra.Group{ID: "core", Title: "Core Commands"})
	RootCmd.AddGroup(&cobra.Group{ID: "analysis", Title: "Analysis & Insights"})
	RootCmd.AddGroup(&cobra.Group{ID: "utility", Title: "Utilities"})

	// Global flags
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

	viper.BindPFlag("json", RootCmd.PersistentFlags().Lookup("json"))           //nolint:errcheck // init-time binding, panic on failure is acceptable
	viper.BindPFlag("pretty", RootCmd.PersistentFlags().Lookup("pretty"))       //nolint:errcheck // init-time binding, panic on failure is acceptable
	viper.BindPFlag("events", RootCmd.PersistentFlags().Lookup("events"))       //nolint:errcheck // init-time binding, panic on failure is acceptable
	viper.BindPFlag("read-only", RootCmd.PersistentFlags().Lookup("read-only")) //nolint:errcheck // init-time binding, panic on failure is acceptable
	viper.BindPFlag("dry-run", RootCmd.PersistentFlags().Lookup("dry-run"))     //nolint:errcheck // init-time binding, panic on failure is acceptable
	viper.BindPFlag("confirm", RootCmd.PersistentFlags().Lookup("confirm"))     //nolint:errcheck // init-time binding, panic on failure is acceptable
	viper.BindPFlag("timeout", RootCmd.PersistentFlags().Lookup("timeout"))     //nolint:errcheck // init-time binding, panic on failure is acceptable
	viper.BindPFlag("profile", RootCmd.PersistentFlags().Lookup("profile"))     //nolint:errcheck // init-time binding, panic on failure is acceptable
	viper.BindPFlag("verbose", RootCmd.PersistentFlags().Lookup("verbose"))     //nolint:errcheck // init-time binding, panic on failure is acceptable

	// Version command
	RootCmd.AddCommand(versionCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Use default paths
		viper.AddConfigPath(config.DefaultDir())
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("MONARCH")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	// Explicit bindings for environment variables specified in requirements
	viper.BindEnv("read-only", "MONARCH_READONLY") //nolint:errcheck
	viper.BindEnv("profile", "MONARCH_PROFILE")    //nolint:errcheck
	viper.BindEnv("timeout", "MONARCH_TIMEOUT")    //nolint:errcheck
	viper.BindEnv("config", "MONARCH_CONFIG")      //nolint:errcheck

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of monarch",
	Run: func(cmd *cobra.Command, args []string) {
		if err := writeVersion(cmd.OutOrStdout(), profile, jsonMode, pretty, time.Duration(0)); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), err) //nolint:errcheck // best-effort stderr
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
		env := output.NewEnvelope("version", profileName, output.SchemaVersion, "", versionPayload{
			Version: version.Version,
			Commit:  version.Commit,
			Date:    version.Date,
			BuiltBy: version.BuiltBy,
		}, duration)
		return renderer.RenderSuccess(env)
	}

	fmt.Fprint(out, monarchBanner) //nolint:errcheck // best-effort banner
	fmt.Fprintln(out)              //nolint:errcheck // best-effort banner
	_, err := fmt.Fprintf(out, "monarch version %s (commit: %s, date: %s, built by: %s)\n", version.Version, version.Commit, version.Date, version.BuiltBy)
	return err
}
