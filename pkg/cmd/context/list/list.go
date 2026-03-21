package list

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/config"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/tableprinter"
)

// Options holds the inputs for context list.
type Options struct {
	Config func() (config.Config, error)
}

// NewCmd creates the "context list" command.
func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		Config: f.Config,
	}

	return &cobra.Command{
		Use:     "list",
		Short:   "List all contexts",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return listRun(opts, f)
		},
	}
}

func listRun(opts *Options, f *cmd.Factory) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	contexts := cfg.Contexts()
	current := cfg.CurrentContext()

	if len(contexts) == 0 {
		fmt.Fprintln(f.IOStreams.ErrOut, "No contexts configured. Run 'a7 context create' to add one.")
		return nil
	}

	tp := tableprinter.New(f.IOStreams.Out)
	tp.SetHeaders("CURRENT", "NAME", "SERVER", "GATEWAY-GROUP")

	for _, ctx := range contexts {
		marker := ""
		if ctx.Name == current {
			marker = "*"
		}
		tp.AddRow(marker, ctx.Name, ctx.Server, ctx.GatewayGroup)
	}

	return tp.Render()
}
