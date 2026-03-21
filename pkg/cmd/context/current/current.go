package current

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/config"
	cmd "github.com/api7/a7/pkg/cmd"
)

// Options holds the inputs for context current.
type Options struct {
	Config func() (config.Config, error)
}

// NewCmd creates the "context current" command.
func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		Config: f.Config,
	}

	return &cobra.Command{
		Use:   "current",
		Short: "Show the current context",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return currentRun(opts, f)
		},
	}
}

func currentRun(opts *Options, f *cmd.Factory) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	name := cfg.CurrentContext()
	if name == "" {
		fmt.Fprintln(f.IOStreams.ErrOut, "No current context. Run 'a7 context create' to add one.")
		return nil
	}

	ctx, err := cfg.GetContext(name)
	if err != nil {
		return err
	}

	fmt.Fprintf(f.IOStreams.Out, "Current context: %s\n", ctx.Name)
	fmt.Fprintf(f.IOStreams.Out, "  Server:        %s\n", ctx.Server)
	if ctx.GatewayGroup != "" {
		fmt.Fprintf(f.IOStreams.Out, "  Gateway Group: %s\n", ctx.GatewayGroup)
	}
	if ctx.TLSSkipVerify {
		fmt.Fprintf(f.IOStreams.Out, "  TLS Skip:      true\n")
	}
	return nil
}
