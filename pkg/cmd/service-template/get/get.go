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
)

type Options struct {
	IO     *iostreams.IOStreams
	Client func() (*http.Client, error)
	Config func() (config.Config, error)
	Output string
	ID     string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a service template",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.ID = args[0]
			return actionRun(opts)
		},
	}
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
	body, err := client.Get(fmt.Sprintf("/api/services/template/%s", opts.ID), nil)
	if err != nil {
		return err
	}

	var item api.ServiceTemplate
	if err := json.Unmarshal(body, &item); err != nil {
		return fmt.Errorf("failed to parse get response: %w", err)
	}

	if opts.Output != "" {
		exporter := cmdutil.NewExporter(opts.Output, opts.IO.Out)
		return exporter.Write(item)
	}

	exporter := cmdutil.NewExporter("json", opts.IO.Out)
	return exporter.Write(item)
}
