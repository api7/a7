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
	PluginName   string
	MetadataJSON string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "create [plugin_name]",
		Short: "Create plugin metadata",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.PluginName = args[0]
			}
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.PluginName, "plugin-name", "", "Plugin name")
	c.Flags().StringVarP(&opts.File, "file", "f", "", "Path to JSON/YAML file with resource definition")
	c.Flags().StringVar(&opts.MetadataJSON, "metadata-json", "", "Plugin metadata JSON object")

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

		pluginName := opts.PluginName
		if pluginName == "" {
			if val, ok := payload["plugin_name"]; ok {
				pluginName = fmt.Sprintf("%v", val)
			}
		}
		if pluginName == "" {
			return fmt.Errorf("--plugin-name is required (or provide plugin_name in --file payload)")
		}

		httpClient, err := opts.Client()
		if err != nil {
			return err
		}

		client := api.NewClient(httpClient, cfg.BaseURL())
		body, err := client.Put("/apisix/admin/plugin_metadata/"+pluginName+"?gateway_group_id="+ggID, payload)
		if err != nil {
			return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
		}

		format := opts.Output
		if format == "" {
			format = "json"
		}
		return cmdutil.NewExporter(format, opts.IO.Out).Write(json.RawMessage(body))
	}
	if opts.PluginName == "" {
		return fmt.Errorf("--plugin-name is required")
	}

	httpClient, err := opts.Client()
	if err != nil {
		return err
	}

	metadata := map[string]interface{}{}
	if opts.MetadataJSON != "" {
		if err := json.Unmarshal([]byte(opts.MetadataJSON), &metadata); err != nil {
			return fmt.Errorf("invalid --metadata-json: %w", err)
		}
	}

	bodyReq := api.PluginMetadata{Metadata: metadata}

	client := api.NewClient(httpClient, cfg.BaseURL())
	body, err := client.Put("/apisix/admin/plugin_metadata/"+opts.PluginName+"?gateway_group_id="+ggID, bodyReq)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var created api.PluginMetadata
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
