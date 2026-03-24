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
	File         string
	GatewayGroup string
	ID           string

	URI    string
	Prefix string
	Token  string
	Labels []string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a secret provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.ID = args[0]
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.URI, "uri", "", "Secret provider URI")
	c.Flags().StringVar(&opts.Prefix, "prefix", "", "Secret provider prefix")
	c.Flags().StringVar(&opts.Token, "token", "", "Secret provider token")
	c.Flags().StringSliceVar(&opts.Labels, "labels", nil, "Labels in key=value format")
	c.Flags().StringVarP(&opts.File, "file", "f", "", "Path to JSON/YAML file with resource definition")

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

	if opts.File != "" {
		payload, err := cmdutil.ReadResourceFile(opts.File, opts.IO.In)
		if err != nil {
			return err
		}
		client := api.NewClient(httpClient, cfg.BaseURL())
		body, err := client.Put("/apisix/admin/secret_providers/"+opts.ID+"?gateway_group_id="+ggID, payload)
		if err != nil {
			return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
		}
		format := opts.Output
		if format == "" {
			format = "json"
		}
		return cmdutil.NewExporter(format, opts.IO.Out).Write(json.RawMessage(body))
	}

	labels := make(map[string]string)
	for _, label := range opts.Labels {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return fmt.Errorf("invalid label %q, expected key=value", label)
		}
		labels[parts[0]] = parts[1]
	}

	bodyReq := api.Secret{
		URI:    opts.URI,
		Prefix: opts.Prefix,
		Token:  opts.Token,
	}
	if len(labels) > 0 {
		bodyReq.Labels = labels
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	body, err := client.Put("/apisix/admin/secret_providers/"+opts.ID+"?gateway_group_id="+ggID, bodyReq)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var updated api.Secret
	if err := json.Unmarshal(body, &updated); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	format := opts.Output
	if format == "" {
		format = "json"
	}
	exporter := cmdutil.NewExporter(format, opts.IO.Out)
	return exporter.Write(updated)
}
