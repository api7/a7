package route

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/route/create"
	del "github.com/api7/a7/pkg/cmd/route/delete"
	"github.com/api7/a7/pkg/cmd/route/get"
	"github.com/api7/a7/pkg/cmd/route/list"
	"github.com/api7/a7/pkg/cmd/route/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "route",
		Short:   "Manage runtime routes",
		Aliases: []string{"rt"},
	}

	c.AddCommand(list.NewCmd(f))
	c.AddCommand(get.NewCmd(f))
	c.AddCommand(create.NewCmd(f))
	c.AddCommand(update.NewCmd(f))
	c.AddCommand(del.NewCmd(f))

	return c
}
