package create

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/pkg/api"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmdutil"
	"github.com/api7/a7/pkg/iostreams"
	"github.com/api7/a7/pkg/tableprinter"
)

type Options struct {
	IO          *iostreams.IOStreams
	Client      func() (*http.Client, error)
	Config      func() (config.Config, error)
	Output      string
	Name        string
	Description string
	Labels      []string
	Prefix      string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
	}
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a gateway group",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			return createRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Name, "name", "", "Gateway group name")
	c.Flags().StringVar(&opts.Description, "description", "", "Gateway group description")
	c.Flags().StringArrayVar(&opts.Labels, "labels", nil, "Gateway group label in key=value format (repeatable)")
	c.Flags().StringVar(&opts.Prefix, "prefix", "", "Gateway group route prefix")

	return c
}

func createRun(opts *Options) error {
	if opts.Name == "" {
		return &cmdutil.FlagError{Err: fmt.Errorf("required flag(s) \"name\" not set")}
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	httpClient, err := opts.Client()
	if err != nil {
		return err
	}

	labels := map[string]string{}
	for _, item := range opts.Labels {
		key, value, found := strings.Cut(item, "=")
		if !found || key == "" {
			return &cmdutil.FlagError{Err: fmt.Errorf("invalid --labels value %q, expected key=value", item)}
		}
		labels[key] = value
	}

	request := map[string]interface{}{
		"name":        opts.Name,
		"description": opts.Description,
		"prefix":      opts.Prefix,
	}
	if len(labels) > 0 {
		request["labels"] = labels
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	body, err := client.Post("/api/gateway_groups", request)
	if err != nil {
		return fmt.Errorf("failed to create gateway group: %s", cmdutil.FormatAPIError(err))
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
