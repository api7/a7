package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/version"
	cmd "github.com/api7/a7/pkg/cmd"
)

// NewCmd creates the version command.
func NewCmd(f *cmd.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the a7 CLI version",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintf(f.IOStreams.Out, "a7 version %s (commit: %s, built: %s)\n",
				version.Version, version.Commit, version.Date)
			return nil
		},
	}
}
