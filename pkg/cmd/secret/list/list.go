package list

import (
	"encoding/json"
	"fmt"
	"net/http"

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
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:     "list",
		Short:   "List secret providers",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			opts.Label, _ = c.Flags().GetString("label")
			return actionRun(opts)
		},
	}
	c.Flags().StringVar(&opts.Label, "label", "", "Filter by label (key=value)")

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
	labelKey, labelValue := cmdutil.ParseLabel(opts.Label)
	if labelKey != "" {
		query["label"] = labelKey
	}
	body, err := client.Get("/apisix/admin/secret_providers", query)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var resp api.ListResponse[api.Secret]
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if labelValue != "" {
		filtered := make([]api.Secret, 0)
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
	tp.SetHeaders("ID", "URI", "PREFIX")
	for _, item := range resp.List {
		tp.AddRow(item.ID, item.URI, item.Prefix)
	}

	return tp.Render()
}
