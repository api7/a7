package sync

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/config"
	"github.com/api7/a7/pkg/api"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/config/configutil"
	"github.com/api7/a7/pkg/cmd/config/validate"
	"github.com/api7/a7/pkg/cmdutil"
	"github.com/api7/a7/pkg/iostreams"
)

type Options struct {
	IO     *iostreams.IOStreams
	Client func() (*http.Client, error)
	Config func() (config.Config, error)

	File         string
	DryRun       bool
	Delete       bool
	GatewayGroup string
}

func NewCmdSync(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		IO:     f.IOStreams,
		Client: f.HttpClient,
		Config: f.Config,
		Delete: true,
	}

	c := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize API7 EE with local declarative config",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			if opts.File == "" {
				return &cmdutil.FlagError{Err: fmt.Errorf("required flag \"file\" not set")}
			}
			return syncRun(opts)
		},
	}

	c.Flags().StringVarP(&opts.File, "file", "f", "", "Path to declarative config file (required)")
	c.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would change without applying")
	c.Flags().BoolVar(&opts.Delete, "delete", true, "Delete remote resources not present in local config")
	c.Flags().StringVar(&opts.GatewayGroup, "gateway-group", "", "Gateway group ID (overrides context default)")

	return c
}

func syncRun(opts *Options) error {
	local, err := configutil.ReadConfigFile(opts.File)
	if err != nil {
		return err
	}

	errs := validate.ValidateConfigFile(local)
	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n- %s", strings.Join(errs, "\n- "))
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

	gatewayGroup := opts.GatewayGroup
	if gatewayGroup == "" {
		gatewayGroup = cfg.GatewayGroup()
	}

	remote, err := configutil.FetchRemoteConfig(client, gatewayGroup)
	if err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}

	result, err := configutil.ComputeDiff(local, *remote)
	if err != nil {
		return err
	}

	if opts.DryRun {
		fmt.Fprint(opts.IO.Out, configutil.FormatDiffSummary(result))
		return nil
	}

	if err := applyAllCreates(client, result, gatewayGroup); err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}
	if err := applyAllUpdates(client, result, gatewayGroup); err != nil {
		return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
	}
	if opts.Delete {
		if err := applyAllDeletes(client, result, gatewayGroup); err != nil {
			return fmt.Errorf("%s", cmdutil.FormatAPIError(err))
		}
	}

	fmt.Fprint(opts.IO.Out, formatSyncSummary(result, opts.Delete))
	return nil
}

func applyAllCreates(client *api.Client, result *configutil.DiffResult, gatewayGroup string) error {
	for _, section := range result.Sections() {
		for _, item := range section.Diff.Create {
			if err := putResource(client, section.Name, item.Key, item.Value, gatewayGroup); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyAllUpdates(client *api.Client, result *configutil.DiffResult, gatewayGroup string) error {
	for _, section := range result.Sections() {
		for _, item := range section.Diff.Update {
			if err := putResource(client, section.Name, item.Key, item.Value, gatewayGroup); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyAllDeletes(client *api.Client, result *configutil.DiffResult, gatewayGroup string) error {
	sections := result.Sections()
	for i := len(sections) - 1; i >= 0; i-- {
		for _, item := range sections[i].Diff.Delete {
			if err := deleteResourceWithRetry(client, sections[i].Name, item.Key, gatewayGroup); err != nil {
				return err
			}
		}
	}
	return nil
}

func isStillReferencedError(err error) bool {
	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == 400 && strings.Contains(apiErr.ErrorMsg, "still using it")
}

func deleteResourceWithRetry(client *api.Client, resourceType, key, gatewayGroup string) error {
	const maxRetries = 5
	backoff := 500 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		lastErr = deleteResource(client, resourceType, key, gatewayGroup)
		if lastErr == nil {
			return nil
		}
		if !isStillReferencedError(lastErr) {
			return lastErr
		}
		if attempt < maxRetries {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	return lastErr
}

func putResource(client *api.Client, resourceType, key string, payload map[string]interface{}, gatewayGroup string) error {
	body := cloneMap(payload)
	path, cleanBody, err := putPathAndBody(resourceType, key, body)
	if err != nil {
		return err
	}
	if gatewayGroup != "" {
		path += "?gateway_group_id=" + gatewayGroup
	}
	_, err = client.Put(path, cleanBody)
	return err
}

func deleteResource(client *api.Client, resourceType, key, gatewayGroup string) error {
	path, err := deletePath(resourceType, key)
	if err != nil {
		return err
	}
	query := map[string]string{}
	if gatewayGroup != "" {
		query["gateway_group_id"] = gatewayGroup
	}
	_, err = client.Delete(path, query)
	return err
}

func putPathAndBody(resourceType, key string, payload map[string]interface{}) (string, map[string]interface{}, error) {
	switch resourceType {
	case "routes":
		return fmt.Sprintf("/apisix/admin/routes/%s", key), payload, nil
	case "services":
		return fmt.Sprintf("/apisix/admin/services/%s", key), payload, nil
	case "upstreams":
		return fmt.Sprintf("/apisix/admin/upstreams/%s", key), payload, nil
	case "consumers":
		return "/apisix/admin/consumers", payload, nil
	case "ssl":
		return fmt.Sprintf("/apisix/admin/ssls/%s", key), payload, nil
	case "global_rules":
		return fmt.Sprintf("/apisix/admin/global_rules/%s", key), payload, nil
	case "plugin_configs":
		return fmt.Sprintf("/apisix/admin/plugin_configs/%s", key), payload, nil
	case "consumer_groups":
		return fmt.Sprintf("/apisix/admin/consumer_groups/%s", key), payload, nil
	case "stream_routes":
		return fmt.Sprintf("/apisix/admin/stream_routes/%s", key), payload, nil
	case "protos":
		return fmt.Sprintf("/apisix/admin/protos/%s", key), payload, nil
	case "secrets":
		return fmt.Sprintf("/apisix/admin/secrets/%s", key), payload, nil
	case "plugin_metadata":
		delete(payload, "plugin_name")
		return fmt.Sprintf("/apisix/admin/plugin_metadata/%s", key), payload, nil
	default:
		return "", nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

func deletePath(resourceType, key string) (string, error) {
	switch resourceType {
	case "routes":
		return fmt.Sprintf("/apisix/admin/routes/%s", key), nil
	case "services":
		return fmt.Sprintf("/apisix/admin/services/%s", key), nil
	case "upstreams":
		return fmt.Sprintf("/apisix/admin/upstreams/%s", key), nil
	case "consumers":
		return fmt.Sprintf("/apisix/admin/consumers/%s", key), nil
	case "ssl":
		return fmt.Sprintf("/apisix/admin/ssls/%s", key), nil
	case "global_rules":
		return fmt.Sprintf("/apisix/admin/global_rules/%s", key), nil
	case "plugin_configs":
		return fmt.Sprintf("/apisix/admin/plugin_configs/%s", key), nil
	case "consumer_groups":
		return fmt.Sprintf("/apisix/admin/consumer_groups/%s", key), nil
	case "stream_routes":
		return fmt.Sprintf("/apisix/admin/stream_routes/%s", key), nil
	case "protos":
		return fmt.Sprintf("/apisix/admin/protos/%s", key), nil
	case "secrets":
		return fmt.Sprintf("/apisix/admin/secrets/%s", key), nil
	case "plugin_metadata":
		return fmt.Sprintf("/apisix/admin/plugin_metadata/%s", key), nil
	default:
		return "", fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

func cloneMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func formatSyncSummary(result *configutil.DiffResult, deleteEnabled bool) string {
	var b strings.Builder
	b.WriteString("Sync completed:\n")
	for _, section := range result.Sections() {
		if deleteEnabled {
			fmt.Fprintf(&b, "%s: created=%d updated=%d deleted=%d\n",
				section.Name, section.Diff.CreateCount(), section.Diff.UpdateCount(), section.Diff.DeleteCount())
		} else {
			fmt.Fprintf(&b, "%s: created=%d updated=%d deleted=0\n",
				section.Name, section.Diff.CreateCount(), section.Diff.UpdateCount())
		}
	}
	return b.String()
}
