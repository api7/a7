package list

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/pkg/api"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmdutil"
	"github.com/api7/a7/pkg/iostreams"
)

type Options struct {
	IO           *iostreams.IOStreams
	Client       func() (*http.Client, error)
	Config       func() (config.Config, error)
	Output       string
	GatewayGroup string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
	}

	c := &cobra.Command{
		Use:     "list",
		Short:   "List runtime plugins",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			return actionRun(opts)
		},
	}

	return c
}

func actionRun(opts *Options) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	ggID := opts.GatewayGroup
	if ggID == "" {
		ggID = cfg.GatewayGroup()
	}
	if ggID == "" {
		return fmt.Errorf("gateway group is required; use --gateway-group flag or set a default in context config")
	}

	httpClient, err := opts.Client()
	if err != nil {
		return err
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	body, err := client.Get("/apisix/admin/plugins/list", map[string]string{"gateway_group_id": ggID})
	if err != nil {
		return fmt.Errorf(cmdutil.FormatAPIError(err))
	}

	var plugins []string
	if err := json.Unmarshal(body, &plugins); err != nil {
		return fmt.Errorf("failed to parse plugin list response: %w", err)
	}

	if opts.Output == "json" {
		return cmdutil.NewExporter("json", opts.IO.Out).Write(plugins)
	}

	for _, name := range plugins {
		if _, err := fmt.Fprintln(opts.IO.Out, name); err != nil {
			return err
		}
	}

	return nil
}
