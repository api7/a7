package create

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	Consumer     string
	File         string
	ID           string

	Name        string
	Desc        string
	PluginsJSON string
	Labels      []string
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{IO: f.IOStreams, Client: f.HttpClient, Config: f.Config}
	c := &cobra.Command{
		Use:   "create [id]",
		Short: "Create a credential (id is server-generated if omitted)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			opts.Consumer, _ = c.Flags().GetString("consumer")
			if len(args) > 0 {
				opts.ID = args[0]
			}
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Consumer, "consumer", "", "Consumer username")
	c.Flags().StringVarP(&opts.File, "file", "f", "", "Path to JSON/YAML file with resource definition")
	c.Flags().StringVar(&opts.Name, "name", "", "Credential display name")
	c.Flags().StringVar(&opts.Desc, "desc", "", "Credential description")
	c.Flags().StringVar(&opts.PluginsJSON, "plugins-json", "", "Plugins JSON string")
	c.Flags().StringSliceVar(&opts.Labels, "labels", nil, "Labels in key=value format")

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

	if opts.File != "" {
		payload, err := cmdutil.ReadResourceFile(opts.File, opts.IO.In)
		if err != nil {
			return err
		}

		// Validate the file's id (if present) regardless of whether the
		// positional overrides it: a bogus value in the file is a user error
		// worth surfacing.
		fileID, hasFileID, err := stringField(payload, "id")
		if err != nil {
			return err
		}

		// Positional [id] overrides any id in the file payload.
		id := strings.TrimSpace(opts.ID)
		if id == "" && hasFileID {
			id = fileID
		}
		// The id, if any, is carried in the URL path on PUT. Drop it from the body
		// so the request body stays clean.
		delete(payload, "id")

		if opts.Name != "" {
			payload["name"] = strings.TrimSpace(opts.Name)
		}
		// API7 EE requires a non-empty `name` on credentials. When the caller
		// gave us an id but no explicit name, mirror the id into the name.
		if _, hasName := payload["name"]; !hasName && id != "" {
			payload["name"] = id
		}
		if _, _, err := stringField(payload, "name"); err != nil {
			return err
		}

		httpClient, err := opts.Client()
		if err != nil {
			return err
		}
		client := api.NewClient(httpClient, cfg.BaseURL())
		return submit(client, opts, ggID, id, payload)
	}

	pl := make(map[string]interface{})
	if opts.PluginsJSON != "" {
		if err := json.Unmarshal([]byte(opts.PluginsJSON), &pl); err != nil {
			return fmt.Errorf("invalid --plugins-json: %w", err)
		}
	}

	labels := make(map[string]string)
	for _, label := range opts.Labels {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return fmt.Errorf("invalid label %q, expected key=value", label)
		}
		labels[parts[0]] = parts[1]
	}

	id := strings.TrimSpace(opts.ID)
	name := strings.TrimSpace(opts.Name)
	if name == "" && id != "" {
		// API7 EE requires a non-empty `name`. Mirror the id when the caller
		// did not pass --name explicitly.
		name = id
	}
	bodyReq := api.Credential{Name: name, Desc: opts.Desc}
	if len(pl) > 0 {
		bodyReq.Plugins = pl
	}
	if len(labels) > 0 {
		bodyReq.Labels = labels
	}

	httpClient, err := opts.Client()
	if err != nil {
		return err
	}
	client := api.NewClient(httpClient, cfg.BaseURL())
	return submit(client, opts, ggID, id, bodyReq)
}

func submit(client *api.Client, opts *Options, ggID, id string, body interface{}) error {
	var (
		raw []byte
		err error
	)
	consumer := url.PathEscape(opts.Consumer)
	group := url.QueryEscape(ggID)
	if id != "" {
		raw, err = client.Put(fmt.Sprintf("/apisix/admin/consumers/%s/credentials/%s?gateway_group_id=%s", consumer, url.PathEscape(id), group), body)
	} else {
		raw, err = client.Post(fmt.Sprintf("/apisix/admin/consumers/%s/credentials?gateway_group_id=%s", consumer, group), body)
	}
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	format := opts.Output
	if format == "" {
		format = "json"
	}
	return cmdutil.NewExporter(format, opts.IO.Out).WriteAPIResponse(raw)
}

// stringField pulls a string field out of a parsed file payload. It returns
// the trimmed value, a bool indicating whether the key was present, or an
// error if the key was present but not a non-empty string (after trimming).
// The trimmed value is written back to the payload when present.
func stringField(payload map[string]interface{}, key string) (string, bool, error) {
	raw, ok := payload[key]
	if !ok {
		return "", false, nil
	}
	str, ok := raw.(string)
	if !ok {
		return "", true, fmt.Errorf("credential %s must be a non-empty string", key)
	}
	trimmed := strings.TrimSpace(str)
	if trimmed == "" {
		return "", true, fmt.Errorf("credential %s must be a non-empty string", key)
	}
	payload[key] = trimmed
	return trimmed, true, nil
}
