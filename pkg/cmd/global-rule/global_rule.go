package globalrule

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/global-rule/create"
	del "github.com/api7/a7/pkg/cmd/global-rule/delete"
	"github.com/api7/a7/pkg/cmd/global-rule/export"
	"github.com/api7/a7/pkg/cmd/global-rule/get"
	"github.com/api7/a7/pkg/cmd/global-rule/list"
	"github.com/api7/a7/pkg/cmd/global-rule/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "global-rule",
		Short:   "Manage global rules",
		Aliases: []string{"gr"},
	}

	c.AddCommand(list.NewCmd(f))
	c.AddCommand(get.NewCmd(f))
	c.AddCommand(create.NewCmd(f))
	c.AddCommand(update.NewCmd(f))
	c.AddCommand(del.NewCmd(f))
	c.AddCommand(export.NewCmd(f))

	return c
}
