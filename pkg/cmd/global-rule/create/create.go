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

	ID          string
	PluginsJSON string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a global rule",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.ID, "id", "", "Global rule ID")
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
	if opts.File != "" {
		payload, err := cmdutil.ReadResourceFile(opts.File, opts.IO.In)
		if err != nil {
			return err
		}

		httpClient, err := opts.Client()
		if err != nil {
			return err
		}

		client := api.NewClient(httpClient, cfg.BaseURL())
		var body []byte
		if id, ok := payload["id"]; ok {
			body, err = client.Put(fmt.Sprintf("/apisix/admin/global_rules/%v?gateway_group_id=%s", id, ggID), payload)
		} else {
			body, err = client.Post("/apisix/admin/global_rules?gateway_group_id="+ggID, payload)
		}
		if err != nil {
			return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
		}

		format := opts.Output
		if format == "" {
			format = "json"
		}
		return cmdutil.NewExporter(format, opts.IO.Out).Write(json.RawMessage(body))
	}
	if opts.ID == "" {
		return fmt.Errorf("--id is required")
	}

	httpClient, err := opts.Client()
	if err != nil {
		return err
	}

	plugins := make(map[string]interface{})
	if opts.PluginsJSON != "" {
		if err := json.Unmarshal([]byte(opts.PluginsJSON), &plugins); err != nil {
			return fmt.Errorf("invalid --plugins-json: %w", err)
		}
	}

	bodyReq := api.GlobalRule{
		ID:      opts.ID,
		Plugins: plugins,
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
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
