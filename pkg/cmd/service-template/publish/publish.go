package publish

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
	IO              *iostreams.IOStreams
	Client          func() (*http.Client, error)
	Config          func() (config.Config, error)
	Output          string
	ID              string
	GatewayGroupIDs []string
}

type publishRequest struct {
	GatewayGroupIDs []string `json:"gateway_group_ids"`
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "publish <id>",
		Short: "Publish a service template",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.ID = args[0]
			return actionRun(opts)
		},
	}

	c.Flags().StringSliceVar(&opts.GatewayGroupIDs, "gateway-group-id", nil, "Gateway group ID (repeatable)")

	return c
}

func actionRun(opts *Options) error {
	if len(opts.GatewayGroupIDs) == 0 {
		return fmt.Errorf("required flag(s) \"gateway-group-id\" not set")
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	httpClient, err := opts.Client()
	if err != nil {
		return err
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	body, err := client.Post(
		fmt.Sprintf("/api/services/template/%s/publish", opts.ID),
		publishRequest{GatewayGroupIDs: opts.GatewayGroupIDs},
	)
	if err != nil {
		return err
	}

	var resp map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &resp); err != nil {
			return fmt.Errorf("failed to parse publish response: %w", err)
		}
	}

	if opts.Output != "" {
		exporter := cmdutil.NewExporter(opts.Output, opts.IO.Out)
		if resp == nil {
			return exporter.Write(map[string]string{"message": "published"})
		}
		return exporter.Write(resp)
	}

	if resp != nil {
		exporter := cmdutil.NewExporter("json", opts.IO.Out)
		return exporter.Write(resp)
	}

	_, err = fmt.Fprintf(opts.IO.Out, "Service template %s published\n", opts.ID)
	return err
}
