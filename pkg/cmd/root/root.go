package root

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/config"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/completion"
	configcmd "github.com/api7/a7/pkg/cmd/config"
	"github.com/api7/a7/pkg/cmd/consumer"
	consumergroup "github.com/api7/a7/pkg/cmd/consumer-group"
	"github.com/api7/a7/pkg/cmd/context"
	"github.com/api7/a7/pkg/cmd/credential"
	"github.com/api7/a7/pkg/cmd/debug"
	gatewaygroup "github.com/api7/a7/pkg/cmd/gateway-group"
	globalrule "github.com/api7/a7/pkg/cmd/global-rule"
	"github.com/api7/a7/pkg/cmd/plugin"
	pluginconfig "github.com/api7/a7/pkg/cmd/plugin-config"
	pluginmetadata "github.com/api7/a7/pkg/cmd/plugin-metadata"
	"github.com/api7/a7/pkg/cmd/proto"
	"github.com/api7/a7/pkg/cmd/route"
	"github.com/api7/a7/pkg/cmd/secret"
	"github.com/api7/a7/pkg/cmd/service"
	servicetemplate "github.com/api7/a7/pkg/cmd/service-template"
	"github.com/api7/a7/pkg/cmd/ssl"
	streamroute "github.com/api7/a7/pkg/cmd/stream-route"
	"github.com/api7/a7/pkg/cmd/update"
	"github.com/api7/a7/pkg/cmd/upstream"
	"github.com/api7/a7/pkg/cmd/version"
)

// NewCmd creates the root a7 command.
func NewCmd(f *cmd.Factory, cfg *config.FileConfig) *cobra.Command {
	c := &cobra.Command{
		Use:           "a7",
		Short:         "CLI for API7 Enterprise Edition",
		Long:          "a7 is a command-line interface for the API7 Enterprise Edition API Gateway.\nManage gateway groups, services, routes, upstreams, consumers, SSL, and more.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global persistent flags — these apply to ALL subcommands.
	c.PersistentFlags().String("server", "", "API7 EE server URL (overrides context config)")
	c.PersistentFlags().String("token", "", "API access token (overrides context config)")
	c.PersistentFlags().String("gateway-group", "", "Default gateway group (overrides context config)")
	c.PersistentFlags().StringP("output", "o", "", "Output format: json, yaml (default: table)")

	// Environment variable overrides — applied at init time.
	cobra.OnInitialize(func() {
		if v := os.Getenv("A7_SERVER"); v != "" {
			cfg.SetServerOverride(v)
		}
		if v := os.Getenv("A7_TOKEN"); v != "" {
			cfg.SetTokenOverride(v)
		}
		if v := os.Getenv("A7_GATEWAY_GROUP"); v != "" {
			cfg.SetGatewayGroupOverride(v)
		}
	})

	// Flag overrides — applied before each command runs (highest priority).
	c.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if v, _ := cmd.Flags().GetString("server"); v != "" {
			cfg.SetServerOverride(v)
		}
		if v, _ := cmd.Flags().GetString("token"); v != "" {
			cfg.SetTokenOverride(v)
		}
		if v, _ := cmd.Flags().GetString("gateway-group"); v != "" {
			cfg.SetGatewayGroupOverride(v)
		}
		return nil
	}

	// Register utility subcommands.
	c.AddCommand(version.NewCmd(f))
	c.AddCommand(completion.NewCmd())
	c.AddCommand(context.NewCmd(f))
	c.AddCommand(configcmd.NewCmd(f))
	c.AddCommand(debug.NewCmd(f))
	c.AddCommand(update.NewCmdUpdate(f))

	// Register resource commands.
	c.AddCommand(gatewaygroup.NewCmd(f))
	c.AddCommand(servicetemplate.NewCmd(f))
	c.AddCommand(route.NewCmd(f))
	c.AddCommand(upstream.NewCmd(f))
	c.AddCommand(consumer.NewCmd(f))
	c.AddCommand(ssl.NewCmd(f))
	c.AddCommand(plugin.NewCmd(f))
	c.AddCommand(service.NewCmd(f))
	c.AddCommand(globalrule.NewCmd(f))
	c.AddCommand(streamroute.NewCmd(f))
	c.AddCommand(pluginconfig.NewCmd(f))
	c.AddCommand(pluginmetadata.NewCmd(f))
	c.AddCommand(consumergroup.NewCmd(f))
	c.AddCommand(credential.NewCmd(f))
	c.AddCommand(secret.NewCmd(f))
	c.AddCommand(proto.NewCmd(f))

	return c
}
