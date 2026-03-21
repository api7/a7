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
	Username     string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
	}

	c := &cobra.Command{
		Use:     "delete <username>",
		Short:   "Delete a consumer by username",
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.Username = args[0]
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
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
	_, err = client.Delete("/apisix/admin/consumers/"+opts.Username, map[string]string{"gateway_group_id": ggID})
	if err != nil {
		return fmt.Errorf(cmdutil.FormatAPIError(err))
	}

	_, err = fmt.Fprintf(opts.IO.Out, "consumer %q deleted\n", opts.Username)
	return err
}
