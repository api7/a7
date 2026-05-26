package create

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
	File         string

	PluginsJSON string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a global rule (id is derived from the plugin name)",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			return actionRun(opts)
		},
	}

	c.Flags().StringVarP(&opts.File, "file", "f", "", "Path to JSON/YAML file with resource definition")
	c.Flags().StringVar(&opts.PluginsJSON, "plugins-json", "", "Plugins as JSON map")

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

	if opts.File == "" && opts.PluginsJSON == "" {
		return fmt.Errorf("one of --file or --plugins-json is required")
	}

	httpClient, err := opts.Client()
	if err != nil {
		return err
	}
	client := api.NewClient(httpClient, cfg.BaseURL())

	if opts.File != "" {
		payload, err := cmdutil.ReadResourceFile(opts.File, opts.IO.In)
		if err != nil {
			return err
		}
		if _, ok := payload["id"]; ok {
			return fmt.Errorf(`"id" must not be set in create payload; the id is derived from the plugin name. Use "a7 global-rule update" to modify an existing rule`)
		}

		body, err := client.Post("/apisix/admin/global_rules?gateway_group_id="+ggID, payload)
		if err != nil {
			return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
		}

		format := opts.Output
		if format == "" {
			format = "json"
		}
		return cmdutil.NewExporter(format, opts.IO.Out).Write(json.RawMessage(body))
	}

	plugins := make(map[string]interface{})
	if err := json.Unmarshal([]byte(opts.PluginsJSON), &plugins); err != nil {
		return fmt.Errorf("invalid --plugins-json: %w", err)
	}

	bodyReq := api.GlobalRule{Plugins: plugins}

	body, err := client.Post("/apisix/admin/global_rules?gateway_group_id="+ggID, bodyReq)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var created api.GlobalRule
	if err := json.Unmarshal(body, &created); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	format := opts.Output
	if format == "" {
		format = "json"
	}
	exporter := cmdutil.NewExporter(format, opts.IO.Out)
	return exporter.Write(created)
}
