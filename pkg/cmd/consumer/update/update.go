package update

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
)

type Options struct {
	IO           *iostreams.IOStreams
	Client       func() (*http.Client, error)
	Config       func() (config.Config, error)
	Output       string
	GatewayGroup string

	Username string
	Desc     string
	Labels   []string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
	}

	c := &cobra.Command{
		Use:   "update <username>",
		Short: "Update a consumer by username",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.Username = args[0]
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Desc, "desc", "", "Consumer description")
	c.Flags().StringArrayVar(&opts.Labels, "labels", nil, "Consumer labels in key=value format (repeatable)")

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

	body := api.Consumer{
		Username: opts.Username,
		Desc:     opts.Desc,
		Labels:   parseLabels(opts.Labels),
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	_, err = client.Put("/apisix/admin/consumers/"+opts.Username+"?gateway_group_id="+ggID, body)
	if err != nil {
		return fmt.Errorf(cmdutil.FormatAPIError(err))
	}

	output := opts.Output
	if output == "" {
		output = "json"
	}

	return cmdutil.NewExporter(output, opts.IO.Out).Write(body)
}

func parseLabels(raw []string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	labels := make(map[string]string, len(raw))
	for _, item := range raw {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
			continue
		}
		labels[parts[0]] = ""
	}
	return labels
}
