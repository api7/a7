package gatewaygroup

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/gateway-group/create"
	del "github.com/api7/a7/pkg/cmd/gateway-group/delete"
	"github.com/api7/a7/pkg/cmd/gateway-group/get"
	"github.com/api7/a7/pkg/cmd/gateway-group/list"
	"github.com/api7/a7/pkg/cmd/gateway-group/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "gateway-group",
		Short:   "Manage gateway groups",
		Aliases: []string{"gg"},
	}
	c.AddCommand(list.NewCmd(f))
	c.AddCommand(get.NewCmd(f))
	c.AddCommand(create.NewCmd(f))
	c.AddCommand(update.NewCmd(f))
	c.AddCommand(del.NewCmd(f))
	return c
}
