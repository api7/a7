package consumergroup

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/consumer-group/create"
	del "github.com/api7/a7/pkg/cmd/consumer-group/delete"
	"github.com/api7/a7/pkg/cmd/consumer-group/export"
	"github.com/api7/a7/pkg/cmd/consumer-group/get"
	"github.com/api7/a7/pkg/cmd/consumer-group/list"
	"github.com/api7/a7/pkg/cmd/consumer-group/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "consumer-group",
		Short:   "Manage consumer groups",
		Aliases: []string{"cg"},
	}

	c.AddCommand(list.NewCmd(f))
	c.AddCommand(get.NewCmd(f))
	c.AddCommand(create.NewCmd(f))
	c.AddCommand(update.NewCmd(f))
	c.AddCommand(del.NewCmd(f))
	c.AddCommand(export.NewCmd(f))

	return c
}
