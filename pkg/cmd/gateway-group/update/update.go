package update

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
	IO             *iostreams.IOStreams
	Client         func() (*http.Client, error)
	Config         func() (config.Config, error)
	Output         string
	File           string
	ID             string
	Name           string
	Description    string
	Labels         []string
	Prefix         string
	NameSet        bool
	DescriptionSet bool
	LabelsSet      bool
	PrefixSet      bool
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
	}
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a gateway group",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.ID = args[0]
			opts.Output, _ = c.Flags().GetString("output")
			opts.NameSet = c.Flags().Changed("name")
			opts.DescriptionSet = c.Flags().Changed("description")
			opts.LabelsSet = c.Flags().Changed("labels")
			opts.PrefixSet = c.Flags().Changed("prefix")
			return updateRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Name, "name", "", "Gateway group name")
	c.Flags().StringVar(&opts.Description, "description", "", "Gateway group description")
	c.Flags().StringArrayVar(&opts.Labels, "labels", nil, "Gateway group label in key=value format (repeatable)")
	c.Flags().StringVar(&opts.Prefix, "prefix", "", "Gateway group route prefix")
	c.Flags().StringVarP(&opts.File, "file", "f", "", "Path to JSON/YAML file with resource definition")

	return c
}

func updateRun(opts *Options) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	httpClient, err := opts.Client()
	if err != nil {
		return err
	}

	if opts.File != "" {
		payload, err := cmdutil.ReadResourceFile(opts.File, opts.IO.In)
		if err != nil {
			return err
		}
		client := api.NewClient(httpClient, cfg.BaseURL())
		body, err := client.Put("/api/gateway_groups/"+opts.ID, payload)
		if err != nil {
			return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
		}
		format := opts.Output
		if format == "" {
			format = "json"
		}
		return cmdutil.NewExporter(format, opts.IO.Out).WriteAPIResponse(body)
	}

	labels := map[string]string{}
	if opts.LabelsSet {
		for _, item := range opts.Labels {
			key, value, found := strings.Cut(item, "=")
			if !found || key == "" {
				return &cmdutil.FlagError{Err: fmt.Errorf("invalid --labels value %q, expected key=value", item)}
			}
			labels[key] = value
		}
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	currentBody, err := client.Get(fmt.Sprintf("/api/gateway_groups/%s", opts.ID), nil)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var current api.GatewayGroup
	if err := json.Unmarshal(currentBody, &current); err != nil {
		return fmt.Errorf("failed to decode current gateway group: %w", err)
	}

	request := map[string]interface{}{
		"name":        current.Name,
		"description": current.Description,
	}
	if current.Prefix != "" {
		request["prefix"] = current.Prefix
	}
	if current.Labels != nil {
		request["labels"] = current.Labels
	}
	if opts.NameSet {
		request["name"] = opts.Name
	}
	if opts.DescriptionSet {
		request["description"] = opts.Description
	}
	if opts.PrefixSet {
		request["prefix"] = opts.Prefix
	}
	if opts.LabelsSet {
		request["labels"] = labels
	}

	body, err := client.Put(fmt.Sprintf("/api/gateway_groups/%s", opts.ID), request)
	if err != nil {
		return fmt.Errorf("failed to update gateway group: %s", cmdutil.FormatAPIError(err))
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
