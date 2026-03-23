package diff

import (
	"fmt"
	"net/http"

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

	File   string
	Output string
}

func NewCmdDiff(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
	}

	c := &cobra.Command{
		Use:   "diff",
		Short: "Show difference between local config and API7 EE",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			if opts.File == "" {
				return &cmdutil.FlagError{Err: fmt.Errorf("required flag \"file\" not set")}
			}
			return diffRun(opts)
		},
	}

	c.Flags().StringVarP(&opts.File, "file", "f", "", "Path to declarative config file (required)")
	c.Flags().StringVarP(&opts.Output, "output", "o", "", "Output format: json")

	return c
}

func diffRun(opts *Options) error {
	local, err := configutil.ReadConfigFile(opts.File)
	if err != nil {
		return err
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

	gatewayGroup := cfg.GatewayGroup()

	remote, err := configutil.FetchRemoteConfig(client, gatewayGroup)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	result, err := configutil.ComputeDiff(local, *remote)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		if err := cmdutil.NewExporter("json", opts.IO.Out).Write(result); err != nil {
			return err
		}
	} else {
		fmt.Fprint(opts.IO.Out, configutil.FormatDiffSummary(result))
	}

	if result.HasDifferences() {
		return &cmdutil.SilentError{Err: fmt.Errorf("differences found")}
	}

	return nil
}
