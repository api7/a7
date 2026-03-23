package list

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
	Label        string
	ServiceID    string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:     "list",
		Short:   "List runtime routes",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			opts.Label, _ = c.Flags().GetString("label")
			opts.ServiceID, _ = c.Flags().GetString("service-id")
			return actionRun(opts)
		},
	}
	c.Flags().StringVar(&opts.Label, "label", "", "Filter by label (key=value)")
	c.Flags().StringVar(&opts.ServiceID, "service-id", "", "Filter by service ID (required by API7 EE)")
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
	if opts.ServiceID != "" {
		query["service_id"] = opts.ServiceID
	}
	labelKey, labelValue := cmdutil.ParseLabel(opts.Label)
	if labelKey != "" {
		query["label"] = labelKey
	}
	body, err := client.Get("/apisix/admin/routes", query)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var resp api.ListResponse[api.Route]
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if labelValue != "" {
		filtered := make([]api.Route, 0)
		for _, item := range resp.List {
			if item.Labels != nil && item.Labels[labelKey] == labelValue {
				filtered = append(filtered, item)
			}
		}
		resp.List = filtered
	}

	if opts.Output != "" {
		exporter := cmdutil.NewExporter(opts.Output, opts.IO.Out)
		return exporter.Write(resp.List)
	}

	tp := tableprinter.New(opts.IO.Out)
	tp.SetHeaders("ID", "NAME", "PATHS", "METHODS", "STATUS")
	for _, item := range resp.List {
		paths := strings.Join(item.Paths, ",")
		if paths == "" {
			paths = item.URI
			if paths == "" && len(item.URIs) > 0 {
				paths = strings.Join(item.URIs, ",")
			}
		}
		tp.AddRow(item.ID, item.Name, paths, strings.Join(item.Methods, ","), fmt.Sprintf("%d", item.Status))
	}

	return tp.Render()
}
