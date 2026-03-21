package export

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

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
	GatewayGroup string
	Label        string
	Output       string
	File         string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "export",
		Short: "Export services as JSON or YAML",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			return actionRun(opts)
		},
	}
	c.Flags().StringVar(&opts.Label, "label", "", "Filter by label (key=value)")
	c.Flags().StringVarP(&opts.Output, "output", "o", "yaml", "Output format: json, yaml")
	c.Flags().StringVarP(&opts.File, "file", "f", "", "Write output to file")
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
	items, err := fetchAll(client, ggID, opts.Label)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	if len(items) == 0 {
		fmt.Fprintln(opts.IO.ErrOut, "No services found.")
		return nil
	}

	format := opts.Output
	if format == "" {
		format = "yaml"
	}

	var out io.Writer = opts.IO.Out
	if opts.File != "" {
		f, err := os.Create(opts.File)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer f.Close()
		out = f
	}

	return cmdutil.NewExporter(format, out).Write(stripTimestamps(items))
}

func fetchAll(client *api.Client, ggID, label string) ([]api.Service, error) {
	page := 1
	pageSize := 100
	var all []api.Service
	labelKey, labelValue := cmdutil.ParseLabel(label)

	for {
		query := map[string]string{
			"gateway_group_id": ggID,
			"page":             fmt.Sprintf("%d", page),
			"page_size":        fmt.Sprintf("%d", pageSize),
		}
		if labelKey != "" {
			query["label"] = labelKey
		}

		body, err := client.Get("/apisix/admin/services", query)
		if err != nil {
			return nil, err
		}

		var resp api.ListResponse[api.Service]
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		for _, item := range resp.List {
			if labelValue != "" && (item.Labels == nil || item.Labels[labelKey] != labelValue) {
				continue
			}
			all = append(all, item)
		}

		if len(resp.List) == 0 || page*pageSize >= resp.Total {
			break
		}
		page++
	}

	return all, nil
}

func stripTimestamps(items []api.Service) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		var m map[string]interface{}
		b, _ := json.Marshal(item)
		_ = json.Unmarshal(b, &m)
		delete(m, "create_time")
		delete(m, "update_time")
		out = append(out, m)
	}
	return out
}
