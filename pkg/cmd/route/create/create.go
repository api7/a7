package create

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

	Name       string
	URI        string
	Methods    []string
	Host       string
	ServiceID  string
	UpstreamID string
	Labels     []string
	Status     int
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config, Status: 1}
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a runtime route",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Name, "name", "", "Route name")
	c.Flags().StringVar(&opts.URI, "uri", "", "Route URI")
	c.Flags().StringSliceVar(&opts.Methods, "methods", nil, "Allowed HTTP methods")
	c.Flags().StringVar(&opts.Host, "host", "", "Route host")
	c.Flags().StringVar(&opts.ServiceID, "service-id", "", "Bound service ID")
	c.Flags().StringVar(&opts.UpstreamID, "upstream-id", "", "Bound upstream ID")
	c.Flags().StringSliceVar(&opts.Labels, "labels", nil, "Labels in key=value format")
	c.Flags().IntVar(&opts.Status, "status", 1, "Route status")

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
	if opts.URI == "" {
		return fmt.Errorf("--uri is required")
	}

	httpClient, err := opts.Client()
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

	bodyReq := api.Route{
		Name:       opts.Name,
		URI:        opts.URI,
		Methods:    opts.Methods,
		Host:       opts.Host,
		ServiceID:  opts.ServiceID,
		UpstreamID: opts.UpstreamID,
		Labels:     labels,
		Status:     opts.Status,
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	body, err := client.Post("/apisix/admin/routes?gateway_group_id="+ggID, bodyReq)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var created api.Route
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
