package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thedavidweng/monarchmoney-cli/internal/config"
	"github.com/thedavidweng/monarchmoney-cli/internal/version"
)

var (
	cfgFile  string
	jsonMode bool
	pretty   bool
	compact  bool
	full     bool
	events   bool
	readOnly bool
	dryRun   bool
	confirm  bool
	timeout  time.Duration
	profile  string
	noColor  bool
	verbose  bool
	debug    bool
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "monarch",
	Short:   "A local, agent-friendly CLI for Monarch Money",
	Version: version.Version,
	Long: `monarchmoney-cli is a single-binary command line tool for working with 
Monarch Money data from your terminal, scripts, and local agents.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Update variables from viper (which handles env vars)
		jsonMode = viper.GetBool("json")
		pretty = viper.GetBool("pretty")
		compact = viper.GetBool("compact")
		full = viper.GetBool("full")
		events = viper.GetBool("events")
		readOnly = viper.GetBool("read-only")
		dryRun = viper.GetBool("dry-run")
		confirm = viper.GetBool("confirm")
		timeout = viper.GetDuration("timeout")
		profile = viper.GetString("profile")
		noColor = viper.GetBool("no-color")
		verbose = viper.GetBool("verbose")
		debug = viper.GetBool("debug")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.monarchmoney-cli/config.yaml)")
	RootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "emit machine-readable JSON")
	RootCmd.PersistentFlags().BoolVar(&pretty, "pretty", false, "pretty-print JSON output")
	RootCmd.PersistentFlags().BoolVar(&compact, "compact", false, "return compact output fields")
	RootCmd.PersistentFlags().BoolVar(&full, "full", false, "return full normalized output fields")
	RootCmd.PersistentFlags().BoolVar(&events, "events", false, "emit NDJSON progress events for long-running commands")
	RootCmd.PersistentFlags().BoolVar(&readOnly, "read-only", false, "block remote writes")
	RootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "preview a remote write without executing it")
	RootCmd.PersistentFlags().BoolVar(&confirm, "confirm", false, "explicitly execute a remote write")
	RootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 30*time.Second, "set command timeout")
	RootCmd.PersistentFlags().StringVar(&profile, "profile", "default", "use a named profile")
	RootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	RootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "print more diagnostics to stderr")
	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "print debug diagnostics to stderr with secrets redacted")

	viper.BindPFlag("json", RootCmd.PersistentFlags().Lookup("json"))
	viper.BindPFlag("pretty", RootCmd.PersistentFlags().Lookup("pretty"))
	viper.BindPFlag("compact", RootCmd.PersistentFlags().Lookup("compact"))
	viper.BindPFlag("full", RootCmd.PersistentFlags().Lookup("full"))
	viper.BindPFlag("events", RootCmd.PersistentFlags().Lookup("events"))
	viper.BindPFlag("read-only", RootCmd.PersistentFlags().Lookup("read-only"))
	viper.BindPFlag("dry-run", RootCmd.PersistentFlags().Lookup("dry-run"))
	viper.BindPFlag("confirm", RootCmd.PersistentFlags().Lookup("confirm"))
	viper.BindPFlag("timeout", RootCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag("profile", RootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("no-color", RootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("verbose", RootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("debug", RootCmd.PersistentFlags().Lookup("debug"))

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
	viper.BindEnv("read-only", "MONARCH_READONLY")
	viper.BindEnv("profile", "MONARCH_PROFILE")
	viper.BindEnv("timeout", "MONARCH_TIMEOUT")
	viper.BindEnv("config", "MONARCH_CONFIG")

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
		fmt.Printf("monarch version %s (commit: %s, date: %s)\n", version.Version, version.Commit, version.Date)
	},
}
