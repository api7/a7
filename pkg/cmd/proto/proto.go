package proto

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/proto/create"
	del "github.com/api7/a7/pkg/cmd/proto/delete"
	"github.com/api7/a7/pkg/cmd/proto/export"
	"github.com/api7/a7/pkg/cmd/proto/get"
	"github.com/api7/a7/pkg/cmd/proto/list"
	"github.com/api7/a7/pkg/cmd/proto/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "proto",
		Short:   "Manage protobuf definitions",
		Aliases: []string{"pb"},
	}

	c.AddCommand(list.NewCmd(f))
	c.AddCommand(get.NewCmd(f))
	c.AddCommand(create.NewCmd(f))
	c.AddCommand(update.NewCmd(f))
	c.AddCommand(del.NewCmd(f))
	c.AddCommand(export.NewCmd(f))

	return c
}
