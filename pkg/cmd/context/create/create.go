package create

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/pkg/api"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmdutil"
)

// Options holds the inputs for context create.
type Options struct {
	Config func() (config.Config, error)

	Name           string
	Server         string
	Token          string
	GatewayGroup   string
	TLSSkipVerify  bool
	CACert         string
	Use            bool
	SkipValidation bool
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
	c.Flags().BoolVar(&opts.SkipValidation, "skip-validation", false, "Skip connectivity validation")

	_ = c.MarkFlagRequired("server")

	return c
}

func validateContext(ctx config.Context) error {
	httpClient := api.NewAuthenticatedClient(ctx.Token, ctx.TLSSkipVerify, ctx.CACert)
	client := api.NewClient(httpClient, ctx.Server)

	// Test connectivity with lightweight API call
	_, err := client.Get("/api/gateway_groups?page_size=1", nil)
	if err != nil {
		var apiErr *api.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.StatusCode {
			case 401:
				return fmt.Errorf("authentication failed: invalid token")
			case 403:
				return fmt.Errorf("permission denied: check token permissions")
			default:
				return fmt.Errorf("server error: %s", cmdutil.FormatAPIError(err))
			}
		}
		// Network error (DNS, connection refused, timeout, etc.)
		return fmt.Errorf("cannot connect to server: %w", err)
	}

	// If gateway-group is specified, verify it exists
	if ctx.GatewayGroup != "" {
		_, err := client.Get("/api/gateway_groups/"+ctx.GatewayGroup, nil)
		if err != nil {
			var apiErr *api.APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
				return fmt.Errorf("gateway group %q not found", ctx.GatewayGroup)
			}
			return fmt.Errorf("failed to verify gateway group: %w", err)
		}
	}

	return nil
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

	// Validate context before saving (unless --skip-validation is set)
	if !opts.SkipValidation {
		fmt.Fprintf(f.IOStreams.Out, "Validating connection to %s...\n", ctx.Server)
		if err := validateContext(ctx); err != nil {
			return fmt.Errorf("validation failed: %w\nUse --skip-validation to bypass this check", err)
		}
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
