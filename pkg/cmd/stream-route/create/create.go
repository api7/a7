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

	Desc       string
	RemoteAddr string
	ServerAddr string
	ServerPort int
	SNI        string
	UpstreamID string
	Labels     []string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a stream (L4) route",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Desc, "desc", "", "Stream route description")
	c.Flags().StringVar(&opts.RemoteAddr, "remote-addr", "", "Remote address")
	c.Flags().StringVar(&opts.ServerAddr, "server-addr", "", "Server address")
	c.Flags().IntVar(&opts.ServerPort, "server-port", 0, "Server port")
	c.Flags().StringVar(&opts.SNI, "sni", "", "SNI")
	c.Flags().StringVar(&opts.UpstreamID, "upstream-id", "", "Bound upstream ID")
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
	if opts.UpstreamID == "" {
		return fmt.Errorf("--upstream-id is required")
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

	bodyReq := api.StreamRoute{
		Desc:       opts.Desc,
		RemoteAddr: opts.RemoteAddr,
		ServerAddr: opts.ServerAddr,
		ServerPort: opts.ServerPort,
		SNI:        opts.SNI,
		UpstreamID: opts.UpstreamID,
	}
	if len(labels) > 0 {
		bodyReq.Labels = labels
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	body, err := client.Post("/apisix/admin/stream_routes?gateway_group_id="+ggID, bodyReq)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var created api.StreamRoute
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
