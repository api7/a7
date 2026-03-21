package delete

import (
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
	IO           *iostreams.IOStreams
	Client       func() (*http.Client, error)
	Config       func() (config.Config, error)
	GatewayGroup string
	Consumer     string
	ID           string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.ID = args[0]
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			opts.Consumer, _ = c.Flags().GetString("consumer")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Consumer, "consumer", "", "Consumer username")
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

	path := "/apisix/admin/consumers/" + opts.Consumer + "/credentials/" + opts.ID
	client := api.NewClient(httpClient, cfg.BaseURL())
	if _, err := client.Delete(path, map[string]string{"gateway_group_id": ggID}); err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	_, err = fmt.Fprintf(opts.IO.Out, "Credential %q deleted.\n", opts.ID)
	return err
}
