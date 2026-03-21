package upstream

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/upstream/create"
	del "github.com/api7/a7/pkg/cmd/upstream/delete"
	"github.com/api7/a7/pkg/cmd/upstream/export"
	"github.com/api7/a7/pkg/cmd/upstream/get"
	"github.com/api7/a7/pkg/cmd/upstream/list"
	"github.com/api7/a7/pkg/cmd/upstream/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "upstream",
		Short:   "Manage runtime upstreams",
		Aliases: []string{"us"},
	}

	c.AddCommand(list.NewCmd(f))
	c.AddCommand(get.NewCmd(f))
	c.AddCommand(create.NewCmd(f))
	c.AddCommand(update.NewCmd(f))
	c.AddCommand(del.NewCmd(f))
	c.AddCommand(export.NewCmd(f))

	return c
}
