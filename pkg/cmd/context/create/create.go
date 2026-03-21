package create

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/config"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmdutil"
)

// Options holds the inputs for context create.
type Options struct {
	Config func() (config.Config, error)

	Name          string
	Server        string
	Token         string
	GatewayGroup  string
	TLSSkipVerify bool
	CACert        string
	Use           bool
}

// NewCmd creates the "context create" command.
func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		Config: f.Config,
	}

	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new connection context",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.Name = args[0]
			return createRun(opts, f)
		},
	}

	c.Flags().StringVar(&opts.Server, "server", "", "API7 EE server URL (required)")
	c.Flags().StringVar(&opts.Token, "token", "", "API access token")
	c.Flags().StringVar(&opts.GatewayGroup, "gateway-group", "", "Default gateway group")
	c.Flags().BoolVar(&opts.TLSSkipVerify, "tls-skip-verify", false, "Skip TLS certificate verification")
	c.Flags().StringVar(&opts.CACert, "ca-cert", "", "Path to CA certificate")
	c.Flags().BoolVar(&opts.Use, "use", false, "Set as current context after creation")

	_ = c.MarkFlagRequired("server")

	return c
}

func createRun(opts *Options, f *cmd.Factory) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	ctx := config.Context{
		Name:          opts.Name,
		Server:        opts.Server,
		Token:         opts.Token,
		GatewayGroup:  opts.GatewayGroup,
		TLSSkipVerify: opts.TLSSkipVerify,
		CACert:        opts.CACert,
	}

	if err := cfg.AddContext(ctx); err != nil {
		return &cmdutil.FlagError{Err: err}
	}

	if opts.Use {
		if err := cfg.SetCurrentContext(opts.Name); err != nil {
			return err
		}
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Fprintf(f.IOStreams.Out, "Context %q created.\n", opts.Name)
	if opts.Use {
		fmt.Fprintf(f.IOStreams.Out, "Switched to context %q.\n", opts.Name)
	}
	return nil
}
