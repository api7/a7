package pluginconfig

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/plugin-config/create"
	del "github.com/api7/a7/pkg/cmd/plugin-config/delete"
	"github.com/api7/a7/pkg/cmd/plugin-config/get"
	"github.com/api7/a7/pkg/cmd/plugin-config/list"
	"github.com/api7/a7/pkg/cmd/plugin-config/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "plugin-config",
		Short:   "Manage reusable plugin configurations",
		Aliases: []string{"pc"},
	}

	c.AddCommand(list.NewCmd(f))
	c.AddCommand(get.NewCmd(f))
	c.AddCommand(create.NewCmd(f))
	c.AddCommand(update.NewCmd(f))
	c.AddCommand(del.NewCmd(f))

	return c
}
