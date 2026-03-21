package use

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/config"
	cmd "github.com/api7/a7/pkg/cmd"
)

// Options holds the inputs for context use.
type Options struct {
	Config func() (config.Config, error)
	Name   string
}

// NewCmd creates the "context use" command.
func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		Config: f.Config,
	}

	return &cobra.Command{
		Use:   "use <name>",
		Short: "Switch to a different context",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.Name = args[0]
			return useRun(opts, f)
		},
	}
}

func useRun(opts *Options, f *cmd.Factory) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	if err := cfg.SetCurrentContext(opts.Name); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Fprintf(f.IOStreams.Out, "Switched to context %q.\n", opts.Name)
	return nil
}
