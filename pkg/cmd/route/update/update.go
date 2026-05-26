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

	Name        string
	Desc        string
	DescSet     bool
	URI         string
	Methods     []string
	Host        string
	ServiceID   string
	Labels      []string
	Status      int
	Priority    int
	StatusSet   bool
	PrioritySet bool
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a runtime route",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.ID = args[0]
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			opts.StatusSet = c.Flags().Changed("status")
			opts.PrioritySet = c.Flags().Changed("priority")
			opts.DescSet = c.Flags().Changed("desc")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Name, "name", "", "Route name")
	c.Flags().StringVar(&opts.Desc, "desc", "", "Route description (pass empty string to clear)")
	c.Flags().StringVar(&opts.URI, "uri", "", "Route URI")
	c.Flags().StringSliceVar(&opts.Methods, "methods", nil, "Allowed HTTP methods")
	c.Flags().StringVar(&opts.Host, "host", "", "Route host")
	c.Flags().StringVar(&opts.ServiceID, "service-id", "", "Bound service ID")
	c.Flags().StringSliceVar(&opts.Labels, "labels", nil, "Labels in key=value format")
	c.Flags().IntVar(&opts.Status, "status", 0, "Route status")
	c.Flags().IntVar(&opts.Priority, "priority", 0, "Route priority")
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
		body, err := client.Put("/apisix/admin/routes/"+opts.ID+"?gateway_group_id="+ggID, payload)
		if err != nil {
			return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
		}
		format := opts.Output
		if format == "" {
			format = "json"
		}
		return cmdutil.NewExporter(format, opts.IO.Out).WriteAPIResponse(body)
	}

	labels := make(map[string]string)
	for _, label := range opts.Labels {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return fmt.Errorf("invalid label %q, expected key=value", label)
		}
		labels[parts[0]] = parts[1]
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	currentBody, err := client.Get("/apisix/admin/routes/"+opts.ID, map[string]string{"gateway_group_id": ggID})
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}
	var bodyReq api.Route
	if err := json.Unmarshal(currentBody, &bodyReq); err != nil {
		return fmt.Errorf("failed to decode current route: %w", err)
	}

	if opts.Name != "" {
		bodyReq.Name = opts.Name
	}
	// DescSet lets the user explicitly clear the description with --desc "".
	if opts.DescSet {
		bodyReq.Desc = opts.Desc
	}
	if opts.URI != "" {
		bodyReq.URI = ""
		bodyReq.URIs = nil
		bodyReq.Paths = []string{opts.URI}
	}
	if len(opts.Methods) > 0 {
		bodyReq.Methods = opts.Methods
	}
	if opts.Host != "" {
		bodyReq.Host = opts.Host
	}
	if opts.ServiceID != "" {
		bodyReq.ServiceID = opts.ServiceID
	}
	if opts.StatusSet {
		bodyReq.Status = opts.Status
	}
	if opts.PrioritySet {
		bodyReq.Priority = opts.Priority
	}
	if len(labels) > 0 {
		bodyReq.Labels = labels
	}

	payload, err := routePayload(bodyReq, opts)
	if err != nil {
		return err
	}
	body, err := client.Put("/apisix/admin/routes/"+opts.ID+"?gateway_group_id="+ggID, payload)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var updated api.Route
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

func routePayload(route api.Route, opts *Options) (interface{}, error) {
	if !opts.StatusSet && !opts.PrioritySet {
		return route, nil
	}

	b, err := json.Marshal(route)
	if err != nil {
		return nil, fmt.Errorf("failed to encode route payload: %w", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, fmt.Errorf("failed to prepare route payload: %w", err)
	}
	if opts.StatusSet {
		payload["status"] = opts.Status
	}
	if opts.PrioritySet {
		payload["priority"] = opts.Priority
	}
	return payload, nil
}
