package pluginmetadata

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/plugin-metadata/create"
	del "github.com/api7/a7/pkg/cmd/plugin-metadata/delete"
	"github.com/api7/a7/pkg/cmd/plugin-metadata/get"
	"github.com/api7/a7/pkg/cmd/plugin-metadata/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "plugin-metadata",
		Short:   "Manage plugin metadata",
		Aliases: []string{"pm"},
	}

	c.AddCommand(get.NewCmd(f))
	c.AddCommand(create.NewCmd(f))
	c.AddCommand(update.NewCmd(f))
	c.AddCommand(del.NewCmd(f))

	return c
}
