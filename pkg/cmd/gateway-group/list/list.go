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
	IO     *iostreams.IOStreams
	Client func() (*http.Client, error)
	Config func() (config.Config, error)
	Output string
	Label  string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
	}
	c := &cobra.Command{
		Use:     "list",
		Short:   "List all gateway groups",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.Label, _ = c.Flags().GetString("label")
			return listRun(opts)
		},
	}
	c.Flags().StringVar(&opts.Label, "label", "", "Filter by label (key=value)")
	return c
}

func listRun(opts *Options) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	httpClient, err := opts.Client()
	if err != nil {
		return err
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	var query map[string]string
	labelKey, labelValue := cmdutil.ParseLabel(opts.Label)
	if labelKey != "" {
		query = map[string]string{"label": labelKey}
	}
	body, err := client.Get("/api/gateway_groups", query)
	if err != nil {
		return fmt.Errorf("failed to list gateway groups: %s", cmdutil.FormatAPIError(err))
	}

	if opts.Output == "json" || opts.Output == "yaml" {
		exp := cmdutil.NewExporter(opts.Output, opts.IO.Out)
		var result api.ListResponse[api.GatewayGroup]
		if err := json.Unmarshal(body, &result); err != nil {
			return err
		}
		return exp.Write(result.List)
	}

	var result api.ListResponse[api.GatewayGroup]
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if labelValue != "" {
		filtered := make([]api.GatewayGroup, 0)
		for _, item := range result.List {
			if item.Labels != nil && item.Labels[labelKey] == labelValue {
				filtered = append(filtered, item)
			}
		}
		result.List = filtered
	}

	tp := tableprinter.New(opts.IO.Out)
	tp.SetHeaders("ID", "NAME", "DESCRIPTION", "STATUS")
	for _, g := range result.List {
		tp.AddRow(g.ID, g.Name, g.Description, fmt.Sprintf("%d", g.Status))
	}
	return tp.Render()
}
