package credential

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/credential/create"
	del "github.com/api7/a7/pkg/cmd/credential/delete"
	"github.com/api7/a7/pkg/cmd/credential/get"
	"github.com/api7/a7/pkg/cmd/credential/list"
	"github.com/api7/a7/pkg/cmd/credential/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "credential",
		Short:   "Manage consumer credentials",
		Aliases: []string{"cred"},
	}

	c.AddCommand(list.NewCmd(f))
	c.AddCommand(get.NewCmd(f))
	c.AddCommand(create.NewCmd(f))
	c.AddCommand(update.NewCmd(f))
	c.AddCommand(del.NewCmd(f))

	return c
}
