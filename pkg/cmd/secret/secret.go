package secret

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/secret/create"
	del "github.com/api7/a7/pkg/cmd/secret/delete"
	"github.com/api7/a7/pkg/cmd/secret/get"
	"github.com/api7/a7/pkg/cmd/secret/list"
	"github.com/api7/a7/pkg/cmd/secret/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "secret",
		Short:   "Manage secret providers",
		Aliases: []string{"sec"},
	}

	c.AddCommand(list.NewCmd(f))
	c.AddCommand(get.NewCmd(f))
	c.AddCommand(create.NewCmd(f))
	c.AddCommand(update.NewCmd(f))
	c.AddCommand(del.NewCmd(f))

	return c
}
