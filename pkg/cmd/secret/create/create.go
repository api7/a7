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
)

type Options struct {
	IO           *iostreams.IOStreams
	Client       func() (*http.Client, error)
	Config       func() (config.Config, error)
	Output       string
	GatewayGroup string
	File         string

	ID     string
	URI    string
	Prefix string
	Token  string
	Labels []string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "create [provider/id]",
		Short: "Create a secret provider",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				if opts.ID != "" && opts.ID != args[0] {
					return fmt.Errorf("positional secret provider id %q conflicts with --id %q", args[0], opts.ID)
				}
				if opts.ID == "" {
					opts.ID = args[0]
				}
			}
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.ID, "id", "", "Secret provider ID")
	c.Flags().StringVarP(&opts.File, "file", "f", "", "Path to JSON/YAML file with resource definition")
	c.Flags().StringVar(&opts.URI, "uri", "", "Secret provider URI")
	c.Flags().StringVar(&opts.Prefix, "prefix", "", "Secret provider prefix")
	c.Flags().StringVar(&opts.Token, "provider-token", "", "Secret provider token")
	c.Flags().StringSliceVar(&opts.Labels, "labels", nil, "Labels in key=value format")

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
	if opts.File != "" {
		payload, err := cmdutil.ReadResourceFile(opts.File, opts.IO.In)
		if err != nil {
			return err
		}
		if opts.ID != "" {
			id := strings.TrimSpace(opts.ID)
			if id == "" {
				return fmt.Errorf("secret provider id is required; use a positional arg or --id")
			}
			payload["id"] = id
		} else {
			id, ok := payload["id"]
			if !ok || id == nil {
				return fmt.Errorf("secret provider id is required; use a positional arg or --id")
			}
			trimmedID := strings.TrimSpace(fmt.Sprint(id))
			if trimmedID == "" {
				return fmt.Errorf("secret provider id is required; use a positional arg or --id")
			}
			payload["id"] = trimmedID
		}

		httpClient, err := opts.Client()
		if err != nil {
			return err
		}

		client := api.NewClient(httpClient, cfg.BaseURL())
		var body []byte
		if id, ok := payload["id"]; ok {
			body, err = client.Put(fmt.Sprintf("/apisix/admin/secret_providers/%v?gateway_group_id=%s", id, ggID), payload)
		} else {
			body, err = client.Post("/apisix/admin/secret_providers?gateway_group_id="+ggID, payload)
		}
		if err != nil {
			return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
		}
		var created api.Secret
		if err := json.Unmarshal(body, &created); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		format := opts.Output
		if format == "" {
			format = "json"
		}
		return cmdutil.NewExporter(format, opts.IO.Out).Write(api.RedactSecret(created))
	}
	if opts.ID == "" {
		return fmt.Errorf("secret provider id is required; use a positional arg or --id")
	}

	httpClient, err := opts.Client()
	if err != nil {
		return err
	}

	labels := make(map[string]string)
	for _, label := range opts.Labels {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return fmt.Errorf("invalid label %q, expected key=value", label)
		}
		labels[parts[0]] = parts[1]
	}

	bodyReq := api.Secret{
		ID:     opts.ID,
		URI:    opts.URI,
		Prefix: opts.Prefix,
		Token:  opts.Token,
	}
	if len(labels) > 0 {
		bodyReq.Labels = labels
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	body, err := client.Put("/apisix/admin/secret_providers/"+opts.ID+"?gateway_group_id="+ggID, bodyReq)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	var created api.Secret
	if err := json.Unmarshal(body, &created); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	format := opts.Output
	if format == "" {
		format = "json"
	}
	exporter := cmdutil.NewExporter(format, opts.IO.Out)
	return exporter.Write(api.RedactSecret(created))
}
