package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/pkg/api"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmdutil"
	"github.com/api7/a7/pkg/iostreams"
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

type Options struct {
	IO     *iostreams.IOStreams
	Client func() (*http.Client, error)
	Config func() (config.Config, error)

	File string
}

func NewCmdValidate(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
	}

	c := &cobra.Command{
		Use:   "validate",
		Short: "Validate a declarative configuration file",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			if opts.File == "" {
				return &cmdutil.FlagError{Err: fmt.Errorf("required flag \"file\" not set")}
			}
			return validateRun(opts)
		},
	}

	c.Flags().StringVarP(&opts.File, "file", "f", "", "Path to declarative config file (required)")

	return c
}

func validateRun(opts *Options) error {
	data, err := readFile(opts.File)
	if err != nil {
		return err
	}

	var cfg api.ConfigFile
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		if err := json.Unmarshal(trimmed, &cfg); err != nil {
			return fmt.Errorf("failed to parse JSON file: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(trimmed, &cfg); err != nil {
			return fmt.Errorf("failed to parse YAML file: %w", err)
		}
	}

	errs := ValidateConfigFile(cfg)
	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n- %s", strings.Join(errs, "\n- "))
	}

	fmt.Fprintln(opts.IO.Out, "Config is valid")
	return nil
}

func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return data, nil
}

func ValidateConfigFile(cfg api.ConfigFile) []string {
	var errs []string

	if cfg.Version == "" {
		errs = append(errs, "version is required")
	} else if cfg.Version != "1" {
		errs = append(errs, "version must be \"1\"")
	}

	seenRouteIDs := map[string]struct{}{}
	for i, r := range cfg.Routes {
		if !hasRouteURI(r) {
			errs = append(errs, fmt.Sprintf("routes[%d]: either uri or uris is required", i))
		}
		if r.ID != "" {
			if err := checkID(r.ID, "routes", i); err != "" {
				errs = append(errs, err)
			} else if _, ok := seenRouteIDs[r.ID]; ok {
				errs = append(errs, fmt.Sprintf("routes[%d]: duplicate id %q", i, r.ID))
			} else {
				seenRouteIDs[r.ID] = struct{}{}
			}
		}
	}

	seenServiceIDs := map[string]struct{}{}
	for i, item := range cfg.Services {
		if item.ID != "" {
			if err := checkID(item.ID, "services", i); err != "" {
				errs = append(errs, err)
			} else if _, ok := seenServiceIDs[item.ID]; ok {
				errs = append(errs, fmt.Sprintf("services[%d]: duplicate id %q", i, item.ID))
			} else {
				seenServiceIDs[item.ID] = struct{}{}
			}
		}
	}

	seenUpstreamIDs := map[string]struct{}{}
	for i, item := range cfg.Upstreams {
		if item.ID != "" {
			if err := checkID(item.ID, "upstreams", i); err != "" {
				errs = append(errs, err)
			} else if _, ok := seenUpstreamIDs[item.ID]; ok {
				errs = append(errs, fmt.Sprintf("upstreams[%d]: duplicate id %q", i, item.ID))
			} else {
				seenUpstreamIDs[item.ID] = struct{}{}
			}
		}
	}

	seenConsumerUsernames := map[string]struct{}{}
	for i, c := range cfg.Consumers {
		if strings.TrimSpace(c.Username) == "" {
			errs = append(errs, fmt.Sprintf("consumers[%d]: username is required", i))
			continue
		}
		username := strings.TrimSpace(c.Username)
		if !idPattern.MatchString(username) {
			errs = append(errs, fmt.Sprintf("consumers[%d]: invalid username %q", i, username))
		} else if _, ok := seenConsumerUsernames[username]; ok {
			errs = append(errs, fmt.Sprintf("consumers[%d]: duplicate username %q", i, username))
		} else {
			seenConsumerUsernames[username] = struct{}{}
		}
	}

	errs = append(errs, checkDuplicateIDs(cfg.SSL, func(s api.SSL) string { return s.ID }, "ssl")...)
	errs = append(errs, checkDuplicateIDs(cfg.GlobalRules, func(g api.GlobalRule) string { return g.ID }, "global_rules")...)
	errs = append(errs, checkDuplicateIDs(cfg.PluginConfigs, func(p api.PluginConfig) string { return p.ID }, "plugin_configs")...)
	errs = append(errs, checkDuplicateIDs(cfg.ConsumerGroups, func(c api.ConsumerGroup) string { return c.ID }, "consumer_groups")...)
	errs = append(errs, checkDuplicateIDs(cfg.StreamRoutes, func(s api.StreamRoute) string { return s.ID }, "stream_routes")...)
	errs = append(errs, checkDuplicateIDs(cfg.Protos, func(p api.Proto) string { return p.ID }, "protos")...)

	seenSecretIDs := map[string]struct{}{}
	for i, item := range cfg.Secrets {
		if item.ID != "" {
			if err := checkSecretID(item.ID, i); err != "" {
				errs = append(errs, err)
			} else if _, ok := seenSecretIDs[item.ID]; ok {
				errs = append(errs, fmt.Sprintf("secrets[%d]: duplicate id %q", i, item.ID))
			} else {
				seenSecretIDs[item.ID] = struct{}{}
			}
		}
	}

	seenPluginMetadataNames := map[string]struct{}{}
	for i, item := range cfg.PluginMetadata {
		raw, ok := item["plugin_name"]
		if !ok {
			errs = append(errs, fmt.Sprintf("plugin_metadata[%d]: plugin_name is required", i))
			continue
		}
		name, ok := raw.(string)
		if !ok || strings.TrimSpace(name) == "" {
			errs = append(errs, fmt.Sprintf("plugin_metadata[%d]: plugin_name must be a non-empty string", i))
			continue
		}
		if !idPattern.MatchString(name) {
			errs = append(errs, fmt.Sprintf("plugin_metadata[%d]: invalid plugin_name %q", i, name))
		} else if _, ok := seenPluginMetadataNames[name]; ok {
			errs = append(errs, fmt.Sprintf("plugin_metadata[%d]: duplicate plugin_name %q", i, name))
		} else {
			seenPluginMetadataNames[name] = struct{}{}
		}
	}

	return errs
}

func hasRouteURI(r api.Route) bool {
	if strings.TrimSpace(r.URI) != "" {
		return true
	}
	for _, uri := range r.URIs {
		if strings.TrimSpace(uri) != "" {
			return true
		}
	}
	return false
}

func checkID(id, section string, idx int) string {
	if !idPattern.MatchString(id) {
		return fmt.Sprintf("%s[%d]: invalid id %q", section, idx, id)
	}
	return ""
}

func checkSecretID(id string, idx int) string {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return fmt.Sprintf("secrets[%d]: invalid id %q", idx, id)
	}
	if !idPattern.MatchString(parts[0]) || !idPattern.MatchString(parts[1]) {
		return fmt.Sprintf("secrets[%d]: invalid id %q", idx, id)
	}
	return ""
}

func checkDuplicateIDs[T any](items []T, getID func(T) string, section string) []string {
	var errs []string
	seen := map[string]struct{}{}
	for i, item := range items {
		id := getID(item)
		if id == "" {
			continue
		}
		if err := checkID(id, section, i); err != "" {
			errs = append(errs, err)
			continue
		}
		if _, ok := seen[id]; ok {
			errs = append(errs, fmt.Sprintf("%s[%d]: duplicate id %q", section, i, id))
		} else {
			seen[id] = struct{}{}
		}
	}
	return errs
}
