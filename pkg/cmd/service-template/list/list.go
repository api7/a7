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
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:     "list",
		Short:   "List service templates",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
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
	body, err := client.Get("/api/services/template", query)
	if err != nil {
		return err
	}

	var resp api.ListResponse[api.ServiceTemplate]
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse list response: %w", err)
	}

	if labelValue != "" {
		filtered := make([]api.ServiceTemplate, 0)
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

	tbl := tableprinter.New(opts.IO.Out)
	tbl.SetHeaders("ID", "NAME", "DESCRIPTION", "STATUS")
	for _, item := range resp.List {
		tbl.AddRow(item.ID, item.Name, item.Description, fmt.Sprintf("%d", item.Status))
	}

	return tbl.Render()
}
