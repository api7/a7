package dump

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/pkg/api"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/config/configutil"
	"github.com/api7/a7/pkg/cmdutil"
	"github.com/api7/a7/pkg/iostreams"
)

type Options struct {
	IO     *iostreams.IOStreams
	Client func() (*http.Client, error)
	Config func() (config.Config, error)

	Output       string
	File         string
	GatewayGroup string
}

func NewCmdDump(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
		Output: "yaml",
	}

	c := &cobra.Command{
		Use:   "dump",
		Short: "Dump API7 EE resources as declarative configuration",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return dumpRun(opts)
		},
	}

	c.Flags().StringVarP(&opts.Output, "output", "o", "yaml", "Output format: yaml, json")
	c.Flags().StringVarP(&opts.File, "file", "f", "", "Write output to file")
	c.Flags().StringVar(&opts.GatewayGroup, "gateway-group", "", "Gateway group ID (overrides context default)")

	return c
}

func dumpRun(opts *Options) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	httpClient, err := opts.Client()
	if err != nil {
		return err
	}

	client := api.NewClient(httpClient, cfg.BaseURL())

	gatewayGroup := opts.GatewayGroup
	if gatewayGroup == "" {
		gatewayGroup = cfg.GatewayGroup()
	}

	remote, err := configutil.FetchRemoteConfig(client, gatewayGroup)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	format := opts.Output
	if format == "" {
		format = "yaml"
	}

	var out io.Writer = opts.IO.Out
	if opts.File != "" {
		f, err := os.Create(opts.File)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer f.Close()
		out = f
	}

	return cmdutil.NewExporter(format, out).Write(remote)
}
