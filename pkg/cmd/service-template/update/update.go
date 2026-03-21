package update

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
	IO          *iostreams.IOStreams
	Client      func() (*http.Client, error)
	Config      func() (config.Config, error)
	Output      string
	File        string
	ID          string
	Name        string
	Description string
	Labels      []string
	Hosts       []string
	PathPrefix  string
}

type updateRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Upstream    map[string]interface{} `json:"upstream,omitempty"`
	Plugins     map[string]interface{} `json:"plugins,omitempty"`
	Hosts       []string               `json:"hosts,omitempty"`
	PathPrefix  string                 `json:"path_prefix,omitempty"`
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a service template",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.ID = args[0]
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Name, "name", "", "Service template name")
	c.Flags().StringVar(&opts.Description, "description", "", "Service template description")
	c.Flags().StringSliceVar(&opts.Labels, "labels", nil, "Labels in key=value format")
	c.Flags().StringSliceVar(&opts.Hosts, "host", nil, "Host to match (repeatable)")
	c.Flags().StringVar(&opts.PathPrefix, "path-prefix", "", "Path prefix")
	c.Flags().StringVarP(&opts.File, "file", "f", "", "Path to JSON/YAML file with resource definition")

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

	if opts.File != "" {
		payload, err := cmdutil.ReadResourceFile(opts.File, opts.IO.In)
		if err != nil {
			return err
		}
		client := api.NewClient(httpClient, cfg.BaseURL())
		body, err := client.Put("/api/services/template/"+opts.ID, payload)
		if err != nil {
			return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
		}
		format := opts.Output
		if format == "" {
			format = "json"
		}
		return cmdutil.NewExporter(format, opts.IO.Out).Write(json.RawMessage(body))
	}

	labels := make(map[string]string, len(opts.Labels))
	for _, label := range opts.Labels {
		k, v := cmdutil.ParseLabel(label)
		if k == "" {
			return fmt.Errorf("invalid label: %q", label)
		}
		labels[k] = v
	}

	req := updateRequest{
		Name:        opts.Name,
		Description: opts.Description,
		Labels:      labels,
		Hosts:       opts.Hosts,
		PathPrefix:  opts.PathPrefix,
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	body, err := client.Put(fmt.Sprintf("/api/services/template/%s", opts.ID), req)
	if err != nil {
		return err
	}

	var item api.ServiceTemplate
	if err := json.Unmarshal(body, &item); err != nil {
		return fmt.Errorf("failed to parse update response: %w", err)
	}

	if opts.Output != "" {
		exporter := cmdutil.NewExporter(opts.Output, opts.IO.Out)
		return exporter.Write(item)
	}

	exporter := cmdutil.NewExporter("json", opts.IO.Out)
	return exporter.Write(item)
}
