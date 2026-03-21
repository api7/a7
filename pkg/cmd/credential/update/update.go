package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	Consumer     string
	ID           string

	Desc        string
	PluginsJSON string
	Labels      []string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.ID = args[0]
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			opts.Consumer, _ = c.Flags().GetString("consumer")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Consumer, "consumer", "", "Consumer username")
	c.Flags().StringVar(&opts.Desc, "desc", "", "Credential description")
	c.Flags().StringVar(&opts.PluginsJSON, "plugins-json", "", "Plugins JSON string")
	c.Flags().StringSliceVar(&opts.Labels, "labels", nil, "Labels in key=value format")
	_ = c.MarkFlagRequired("consumer")

	return c
}

func actionRun(opts *Options) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	if opts.Consumer == "" {
		return fmt.Errorf("--consumer is required")
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

	pl := make(map[string]interface{})
	if opts.PluginsJSON != "" {
		if err := json.Unmarshal([]byte(opts.PluginsJSON), &pl); err != nil {
			return fmt.Errorf("invalid --plugins-json: %w", err)
		}
	}

	labels := make(map[string]string)
	for _, label := range opts.Labels {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return fmt.Errorf("invalid label %q, expected key=value", label)
		}
		labels[parts[0]] = parts[1]
	}

	bodyReq := api.Credential{ID: opts.ID, Desc: opts.Desc}
	if len(pl) > 0 {
		bodyReq.Plugins = pl
	}
	if len(labels) > 0 {
		bodyReq.Labels = labels
	}

	path := "/apisix/admin/consumers/" + opts.Consumer + "/credentials/" + opts.ID + "?gateway_group_id=" + ggID
	client := api.NewClient(httpClient, cfg.BaseURL())
	body, err := client.Put(path, bodyReq)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var updated api.Credential
	if err := json.Unmarshal(body, &updated); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	format := opts.Output
	if format == "" {
		format = "json"
	}
	return cmdutil.NewExporter(format, opts.IO.Out).Write(updated)
}
