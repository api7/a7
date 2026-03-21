package completion

import (
	"os"

	"github.com/spf13/cobra"
)

// NewCmd creates the completion command.
func NewCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for a7.

To load completions:

  Bash:
    $ source <(a7 completion bash)
    # To install permanently:
    $ a7 completion bash > /etc/bash_completion.d/a7

  Zsh:
    $ source <(a7 completion zsh)
    # To install permanently:
    $ a7 completion zsh > "${fpath[1]}/_a7"

  Fish:
    $ a7 completion fish | source
    # To install permanently:
    $ a7 completion fish > ~/.config/fish/completions/a7.fish

  PowerShell:
    PS> a7 completion powershell | Out-String | Invoke-Expression
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
	return c
}
