package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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
	ID           string

	Name         string
	Type         string
	Nodes        []string
	Scheme       string
	Retries      int
	PassHost     string
	UpstreamHost string
	Labels       []string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a runtime upstream",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.ID = args[0]
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Name, "name", "", "Upstream name")
	c.Flags().StringVar(&opts.Type, "type", "", "Load balancing type")
	c.Flags().StringSliceVar(&opts.Nodes, "nodes", nil, "Upstream node, repeatable, format host:port=weight")
	c.Flags().StringVar(&opts.Scheme, "scheme", "", "Upstream scheme")
	c.Flags().IntVar(&opts.Retries, "retries", 0, "Retry count")
	c.Flags().StringVar(&opts.PassHost, "pass-host", "", "Pass host mode")
	c.Flags().StringVar(&opts.UpstreamHost, "upstream-host", "", "Upstream host override")
	c.Flags().StringSliceVar(&opts.Labels, "labels", nil, "Labels in key=value format")

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

	nodes, err := parseNodes(opts.Nodes)
	if err != nil {
		return err
	}

	labels := make(map[string]string)
	for _, label := range opts.Labels {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return fmt.Errorf("invalid label %q, expected key=value", label)
		}
		labels[parts[0]] = parts[1]
	}

	bodyReq := api.Upstream{
		Name:         opts.Name,
		Type:         opts.Type,
		Scheme:       opts.Scheme,
		Retries:      opts.Retries,
		PassHost:     opts.PassHost,
		UpstreamHost: opts.UpstreamHost,
	}
	if len(nodes) > 0 {
		bodyReq.Nodes = nodes
	}
	if len(labels) > 0 {
		bodyReq.Labels = labels
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	body, err := client.Put("/apisix/admin/upstreams/"+opts.ID+"?gateway_group_id="+ggID, bodyReq)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var updated api.Upstream
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

func parseNodes(items []string) (map[string]int, error) {
	out := make(map[string]int)
	for _, item := range items {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("invalid node %q, expected host:port=weight", item)
		}
		weight, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid node %q: invalid weight: %w", item, err)
		}
		out[parts[0]] = weight
	}
	return out, nil
}
