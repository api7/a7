package dump

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

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

	Output        string
	File          string
	LabelSelector []string
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
	c.Flags().StringArrayVar(&opts.LabelSelector, "label-selector", nil,
		"Filter dumped resources by label in key=value format (repeatable; multiple selectors are AND-combined)")

	return c
}

// parseLabelSelectors converts repeated key=value flags into a labels map.
// Whitespace around key and value is preserved (callers typically supply
// shell-quoted values). An empty key, or any entry missing "=", is rejected
// so users get a clear error instead of silently filtering on the wrong field.
func parseLabelSelectors(raw []string) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(raw))
	for _, item := range raw {
		idx := strings.Index(item, "=")
		if idx < 0 {
			return nil, fmt.Errorf("invalid --label-selector %q: expected key=value", item)
		}
		key := item[:idx]
		value := item[idx+1:]
		if key == "" {
			return nil, fmt.Errorf("invalid --label-selector %q: key is empty", item)
		}
		out[key] = value
	}
	return out, nil
}

func dumpRun(opts *Options) error {
	labels, err := parseLabelSelectors(opts.LabelSelector)
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

	remote, err := configutil.FetchRemoteConfigWithLabels(client, gatewayGroup, labels)
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
