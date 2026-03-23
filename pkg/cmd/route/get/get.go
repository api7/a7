package get

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
	"github.com/api7/a7/pkg/tableprinter"
)

type Options struct {
	IO           *iostreams.IOStreams
	Client       func() (*http.Client, error)
	Config       func() (config.Config, error)
	Output       string
	GatewayGroup string
	ID           string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a runtime route",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.ID = args[0]
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
	query := map[string]string{"gateway_group_id": ggID}
	body, err := client.Get("/apisix/admin/routes/"+opts.ID, query)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var item api.Route
	if err := json.Unmarshal(body, &item); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if opts.Output != "" {
		exporter := cmdutil.NewExporter(opts.Output, opts.IO.Out)
		return exporter.Write(item)
	}

	tp := tableprinter.New(opts.IO.Out)
	tp.SetHeaders("FIELD", "VALUE")
	tp.AddRow("id", item.ID)
	tp.AddRow("name", item.Name)
	tp.AddRow("uri", item.URI)
	tp.AddRow("paths", strings.Join(item.Paths, ", "))
	tp.AddRow("host", item.Host)
	tp.AddRow("service_id", item.ServiceID)
	tp.AddRow("upstream_id", item.UpstreamID)
	tp.AddRow("status", fmt.Sprintf("%d", item.Status))
	tp.AddRow("priority", fmt.Sprintf("%d", item.Priority))

	return tp.Render()
}
