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
	IO     *iostreams.IOStreams
	Client func() (*http.Client, error)
	Config func() (config.Config, error)
	ID     string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
	}
	c := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a gateway group",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.ID = args[0]
			return deleteRun(opts)
		},
	}
	return c
}

func deleteRun(opts *Options) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	httpClient, err := opts.Client()
	if err != nil {
		return err
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	if _, err := client.Delete(fmt.Sprintf("/api/gateway_groups/%s", opts.ID), nil); err != nil {
		return fmt.Errorf("failed to delete gateway group: %s", cmdutil.FormatAPIError(err))
	}

	_, err = fmt.Fprintf(opts.IO.Out, "Gateway group %q deleted.\n", opts.ID)
	return err
}
