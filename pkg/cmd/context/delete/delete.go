package delete

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/config"
	cmd "github.com/api7/a7/pkg/cmd"
)

// Options holds the inputs for context delete.
type Options struct {
	Config func() (config.Config, error)
	Name   string
}

// NewCmd creates the "context delete" command.
func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		Config: f.Config,
	}

	return &cobra.Command{
		Use:     "delete <name>",
		Short:   "Delete a context",
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.Name = args[0]
			return deleteRun(opts, f)
		},
	}
}

func deleteRun(opts *Options, f *cmd.Factory) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	if err := cfg.RemoveContext(opts.Name); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Fprintf(f.IOStreams.Out, "Context %q deleted.\n", opts.Name)
	return nil
}
