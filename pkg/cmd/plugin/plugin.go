package plugin

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	pluginlist "github.com/api7/a7/pkg/cmd/plugin/list"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "plugin",
		Aliases: []string{"pl"},
		Short:   "Manage APISIX runtime plugins",
	}

	c.AddCommand(pluginlist.NewCmd(f))

	return c
}
