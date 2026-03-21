package service

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/service/create"
	del "github.com/api7/a7/pkg/cmd/service/delete"
	"github.com/api7/a7/pkg/cmd/service/get"
	"github.com/api7/a7/pkg/cmd/service/list"
	"github.com/api7/a7/pkg/cmd/service/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "service",
		Short:   "Manage runtime services",
		Aliases: []string{"svc"},
	}

	c.AddCommand(list.NewCmd(f))
	c.AddCommand(get.NewCmd(f))
	c.AddCommand(create.NewCmd(f))
	c.AddCommand(update.NewCmd(f))
	c.AddCommand(del.NewCmd(f))

	return c
}
