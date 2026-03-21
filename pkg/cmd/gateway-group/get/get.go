package get

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
	ID     string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
	}
	c := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a gateway group by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.ID = args[0]
			opts.Output, _ = c.Flags().GetString("output")
			return getRun(opts)
		},
	}
	return c
}

func getRun(opts *Options) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	httpClient, err := opts.Client()
	if err != nil {
		return err
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	body, err := client.Get(fmt.Sprintf("/api/gateway_groups/%s", opts.ID), nil)
	if err != nil {
		return fmt.Errorf("failed to get gateway group: %s", cmdutil.FormatAPIError(err))
	}

	var result api.GatewayGroup
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if opts.Output == "json" || opts.Output == "yaml" {
		exp := cmdutil.NewExporter(opts.Output, opts.IO.Out)
		return exp.Write(result)
	}

	tp := tableprinter.New(opts.IO.Out)
	tp.SetHeaders("ID", "NAME", "DESCRIPTION", "PREFIX", "STATUS")
	tp.AddRow(result.ID, result.Name, result.Description, result.Prefix, fmt.Sprintf("%d", result.Status))
	return tp.Render()
}
