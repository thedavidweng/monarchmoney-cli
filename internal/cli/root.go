package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/monarchmoney-cli/monarch/internal/config"
	"github.com/monarchmoney-cli/monarch/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	jsonMode  bool
	pretty    bool
	compact   bool
	full      bool
	events    bool
	readOnly  bool
	dryRun    bool
	confirm   bool
	timeout   time.Duration
	profile   string
	noColor   bool
	verbose   bool
	debug     bool
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "monarch",
	Short:   "A local, agent-friendly CLI for Monarch Money",
	Version: version.Version,
	Long: `monarchmoney-cli is a single-binary command line tool for working with 
Monarch Money data from your terminal, scripts, and local agents.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize config and other global state
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
	viper.AutomaticEnv() // read in environment variables that match

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
