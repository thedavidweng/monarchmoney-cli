package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for monarch.

To load completions:

Bash:

  $ source <(monarch completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ monarch completion bash > /etc/bash_completion.d/monarch
  # macOS:
  $ monarch completion bash > $(brew --prefix)/etc/bash_completion.d/monarch

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ monarch completion zsh > "${fpath[1]}/_monarch"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ monarch completion fish | source

  # To load completions for each session, execute once:
  $ monarch completion fish > ~/.config/fish/completions/monarch.fish

PowerShell:

  PS> monarch completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> monarch completion powershell > monarch.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	GroupID:               "utility",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletionV2(out, true)
		case "zsh":
			return cmd.Root().GenZshCompletion(out)
		case "fish":
			return cmd.Root().GenFishCompletion(out, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(out)
		default:
			return fmt.Errorf("unsupported shell: %q", args[0]) //nolint:gocritic // unreachable but defensive
		}
	},
}

func init() {
	RootCmd.AddCommand(completionCmd)
}
