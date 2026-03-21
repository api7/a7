package create

import (
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
	GatewayGroup string

	Cert   string
	Key    string
	SNIs   []string
	Type   string
	Labels []string
	Status int
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
		Use:   "create",
		Short: "Create an SSL certificate",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			opts.GatewayGroup, _ = c.Flags().GetString("gateway-group")
			return actionRun(opts)
		},
	}

	c.Flags().StringVar(&opts.Cert, "cert", "", "Certificate content or file path")
	c.Flags().StringVar(&opts.Key, "key", "", "Private key content or file path")
	c.Flags().StringArrayVar(&opts.SNIs, "sni", nil, "SNI value (repeatable)")
	c.Flags().StringVar(&opts.Type, "type", "server", "SSL type")
	c.Flags().StringArrayVar(&opts.Labels, "labels", nil, "SSL labels in key=value format (repeatable)")
	c.Flags().IntVar(&opts.Status, "status", 1, "SSL status")

	_ = c.MarkFlagRequired("cert")
	_ = c.MarkFlagRequired("key")
	_ = c.MarkFlagRequired("sni")

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

	cert, err := maybeReadFile(opts.Cert)
	if err != nil {
		return err
	}
	key, err := maybeReadFile(opts.Key)
	if err != nil {
		return err
	}

	body := api.SSL{
		Cert:   cert,
		Key:    key,
		SNIs:   opts.SNIs,
		Labels: parseLabels(opts.Labels),
		Type:   opts.Type,
		Status: opts.Status,
	}

	client := api.NewClient(httpClient, cfg.BaseURL())
	_, err = client.Post("/apisix/admin/ssls?gateway_group_id="+ggID, body)
	if err != nil {
		return fmt.Errorf(cmdutil.FormatAPIError(err))
	}

	output := opts.Output
	if output == "" {
		output = "json"
	}

	return cmdutil.NewExporter(output, opts.IO.Out).Write(body)
}

func maybeReadFile(input string) (string, error) {
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
	return strings.HasPrefix(v, "/") || strings.HasPrefix(v, "./") || strings.HasPrefix(v, "~/")
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
