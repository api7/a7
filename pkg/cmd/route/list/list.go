package list

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/pkg/api"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmdutil"
	"github.com/api7/a7/pkg/iostreams"
	"github.com/api7/a7/pkg/listutil"
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
	c.Flags().StringVar(&opts.ServiceID, "service-id", "", "Filter to routes belonging to a single service; omit to list all routes in the gateway group")
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

	labelKey, labelValue := cmdutil.ParseLabel(opts.Label)
	baseQuery := map[string]string{"gateway_group_id": ggID}

	routes, err := fetchRoutes(client, baseQuery, opts.ServiceID, labelKey)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	if labelValue != "" {
		filtered := make([]api.Route, 0, len(routes))
		for _, item := range routes {
			if item.Labels != nil && item.Labels[labelKey] == labelValue {
				filtered = append(filtered, item)
			}
		}
		routes = filtered
	}

	if opts.Output != "" {
		exporter := cmdutil.NewExporter(opts.Output, opts.IO.Out)
		return exporter.Write(routes)
	}

	tp := tableprinter.New(opts.IO.Out)
	tp.SetHeaders("ID", "NAME", "SERVICE_ID", "PATHS", "METHODS", "STATUS")
	for _, item := range routes {
		paths := strings.Join(item.Paths, ",")
		if paths == "" {
			paths = item.URI
			if paths == "" && len(item.URIs) > 0 {
				paths = strings.Join(item.URIs, ",")
			}
		}
		tp.AddRow(item.ID, item.Name, item.ServiceID, paths, strings.Join(item.Methods, ","), fmt.Sprintf("%d", item.Status))
	}

	return tp.Render()
}

// fetchRoutes returns the paginated route slice for the request. With a
// service ID, it pages through a single filtered query; without one, it lists
// every service in the gateway group and aggregates their routes (API7 EE
// requires `service_id` on this endpoint under access-token auth).
//
// labelKey is applied only to the /routes calls. /services discovery stays
// label-free so that services without the label are still enumerated and
// their matching routes returned.
func fetchRoutes(client *api.Client, baseQuery map[string]string, serviceID, labelKey string) ([]api.Route, error) {
	routeQuery := make(map[string]string, len(baseQuery)+2)
	for k, v := range baseQuery {
		routeQuery[k] = v
	}
	if labelKey != "" {
		routeQuery["label"] = labelKey
	}

	if serviceID != "" {
		routeQuery["service_id"] = serviceID
		return listutil.FetchPaginated[api.Route](client, "/apisix/admin/routes", routeQuery, false)
	}

	services, err := listutil.FetchPaginated[api.Service](client, "/apisix/admin/services", baseQuery, false)
	if err != nil {
		return nil, err
	}
	return listutil.FetchRoutesForServices(client, services, routeQuery, false)
}
