package consumer

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	consumercreate "github.com/api7/a7/pkg/cmd/consumer/create"
	consumerdelete "github.com/api7/a7/pkg/cmd/consumer/delete"
	consumerexport "github.com/api7/a7/pkg/cmd/consumer/export"
	consumerget "github.com/api7/a7/pkg/cmd/consumer/get"
	consumerlist "github.com/api7/a7/pkg/cmd/consumer/list"
	consumerupdate "github.com/api7/a7/pkg/cmd/consumer/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "consumer",
		Aliases: []string{"c"},
		Short:   "Manage APISIX runtime consumers",
	}

	c.AddCommand(consumerlist.NewCmd(f))
	c.AddCommand(consumerget.NewCmd(f))
	c.AddCommand(consumercreate.NewCmd(f))
	c.AddCommand(consumerupdate.NewCmd(f))
	c.AddCommand(consumerdelete.NewCmd(f))
	c.AddCommand(consumerexport.NewCmd(f))

	return c
}
