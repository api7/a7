package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
	File         string
	GatewayGroup string

	ID        string
	Cert      string
	Key       string
	SNIs      []string
	Type      string
	Labels    []string
	Status    int
	TypeSet   bool
	StatusSet bool
}

func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
		Type:   "server",
		Status: 1,
	}

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an SSL certificate by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.ID = args[0]
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			opts.TypeSet = c.Flags().Changed("type")
			opts.StatusSet = c.Flags().Changed("status")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Cert, "cert", "", "Certificate content or file path")
	c.Flags().StringVar(&opts.Key, "key", "", "Private key content or file path")
	c.Flags().StringArrayVar(&opts.SNIs, "sni", nil, "SNI value (repeatable)")
	c.Flags().StringVar(&opts.Type, "type", "server", "SSL type")
	c.Flags().StringArrayVar(&opts.Labels, "labels", nil, "SSL labels in key=value format (repeatable)")
	c.Flags().IntVar(&opts.Status, "status", 1, "SSL status")
	c.Flags().StringVarP(&opts.File, "file", "f", "", "Path to JSON/YAML file with resource definition")

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

	if opts.File != "" {
		payload, err := cmdutil.ReadResourceFile(opts.File, opts.IO.In)
		if err != nil {
			return err
		}
		client := api.NewClient(httpClient, cfg.BaseURL())
		body, err := client.Put("/apisix/admin/ssls/"+opts.ID+"?gateway_group_id="+ggID, payload)
		if err != nil {
			return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
		}
		format := opts.Output
		if format == "" {
			format = "json"
		}
		return cmdutil.NewExporter(format, opts.IO.Out).Write(json.RawMessage(body))
	}

	cert, err := maybeReadFile(opts.Cert)
	if err != nil {
		return err
	}
	key, err := maybeReadFile(opts.Key)
	if err != nil {
		return err
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	currentBody, err := client.Get("/apisix/admin/ssls/"+opts.ID, map[string]string{"gateway_group_id": ggID})
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}
	var body api.SSL
	if err := json.Unmarshal(currentBody, &body); err != nil {
		return fmt.Errorf("failed to decode current ssl: %w", err)
	}

	if cert != "" {
		body.Cert = cert
	}
	if key != "" {
		body.Key = key
	}
	if len(opts.SNIs) > 0 {
		body.SNIs = opts.SNIs
	}
	if len(opts.Labels) > 0 {
		body.Labels = parseLabels(opts.Labels)
	}
	if opts.TypeSet {
		body.Type = opts.Type
	}
	if opts.StatusSet {
		body.Status = opts.Status
	}

	payload, err := sslPayload(body, opts)
	if err != nil {
		return err
	}
	_, err = client.Put("/apisix/admin/ssls/"+opts.ID+"?gateway_group_id="+ggID, payload)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	output := opts.Output
	if output == "" {
		output = "json"
	}

	return cmdutil.NewExporter(output, opts.IO.Out).Write(body)
}

func sslPayload(ssl api.SSL, opts *Options) (interface{}, error) {
	if !opts.StatusSet || opts.Status != 0 {
		return ssl, nil
	}

	b, err := json.Marshal(ssl)
	if err != nil {
		return nil, fmt.Errorf("failed to encode ssl payload: %w", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, fmt.Errorf("failed to prepare ssl payload: %w", err)
	}
	payload["status"] = opts.Status
	return payload, nil
}

func maybeReadFile(input string) (string, error) {
	if input == "" {
		return "", nil
	}
	if !looksLikePath(input) {
		return input, nil
	}
	path := input
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory for %q: %w", input, err)
		}
		path = filepath.Join(home, path[2:])
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %q: %w", path, err)
	}
	return string(b), nil
}

func looksLikePath(v string) bool {
	if strings.Contains(v, "-----BEGIN ") || strings.Contains(v, "\n") {
		return false
	}
	if strings.HasPrefix(v, "/") || strings.HasPrefix(v, "./") || strings.HasPrefix(v, "~/") {
		return true
	}
	info, err := os.Stat(v)
	return err == nil && !info.IsDir()
}

func parseLabels(raw []string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	labels := make(map[string]string, len(raw))
	for _, item := range raw {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
			continue
		}
		labels[parts[0]] = ""
	}
	return labels
}
