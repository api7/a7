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
	Consumer     string
	Label        string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:     "list",
		Short:   "List consumer credentials",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			opts.Consumer, _ = c.Flags().GetString("consumer")
			opts.Label, _ = c.Flags().GetString("label")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Consumer, "consumer", "", "Consumer username")
	c.Flags().StringVar(&opts.Label, "label", "", "Filter by label (key=value)")
	_ = c.MarkFlagRequired("consumer")

	return c
}

func actionRun(opts *Options) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	if opts.Consumer == "" {
		return fmt.Errorf("--consumer is required")
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

	path := "/apisix/admin/consumers/" + opts.Consumer + "/credentials"
	client := api.NewClient(httpClient, cfg.BaseURL())
	query := map[string]string{"gateway_group_id": ggID}
	labelKey, labelValue := cmdutil.ParseLabel(opts.Label)
	if labelKey != "" {
		query["label"] = labelKey
	}
	body, err := client.Get(path, query)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var resp api.ListResponse[api.Credential]
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if labelValue != "" {
		filtered := make([]api.Credential, 0)
		for _, item := range resp.List {
			if item.Labels != nil && item.Labels[labelKey] == labelValue {
				filtered = append(filtered, item)
			}
		}
		resp.List = filtered
	}

	if opts.Output != "" {
		return cmdutil.NewExporter(opts.Output, opts.IO.Out).Write(resp.List)
	}

	tp := tableprinter.New(opts.IO.Out)
	tp.SetHeaders("ID", "DESCRIPTION", "PLUGINS")
	for _, item := range resp.List {
		tp.AddRow(item.ID, item.Desc, fmt.Sprintf("%d", len(item.Plugins)))
	}

	return tp.Render()
}
