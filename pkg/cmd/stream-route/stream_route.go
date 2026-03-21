package streamroute

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/stream-route/create"
	del "github.com/api7/a7/pkg/cmd/stream-route/delete"
	"github.com/api7/a7/pkg/cmd/stream-route/get"
	"github.com/api7/a7/pkg/cmd/stream-route/list"
	"github.com/api7/a7/pkg/cmd/stream-route/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "stream-route",
		Short:   "Manage stream (L4) routes",
		Aliases: []string{"sr"},
	}

	c.AddCommand(list.NewCmd(f))
	c.AddCommand(get.NewCmd(f))
	c.AddCommand(create.NewCmd(f))
	c.AddCommand(update.NewCmd(f))
	c.AddCommand(del.NewCmd(f))

	return c
}
