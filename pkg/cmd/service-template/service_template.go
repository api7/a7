package servicetemplate

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	servicetemplatecreate "github.com/api7/a7/pkg/cmd/service-template/create"
	servicetemplatedelete "github.com/api7/a7/pkg/cmd/service-template/delete"
	servicetemplateget "github.com/api7/a7/pkg/cmd/service-template/get"
	servicetemplatelist "github.com/api7/a7/pkg/cmd/service-template/list"
	servicetemplatepublish "github.com/api7/a7/pkg/cmd/service-template/publish"
	servicetemplateupdate "github.com/api7/a7/pkg/cmd/service-template/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "service-template",
		Short:   "Manage service templates",
		Aliases: []string{"st"},
	}

	c.AddCommand(servicetemplatelist.NewCmd(f))
	c.AddCommand(servicetemplateget.NewCmd(f))
	c.AddCommand(servicetemplatecreate.NewCmd(f))
	c.AddCommand(servicetemplateupdate.NewCmd(f))
	c.AddCommand(servicetemplatedelete.NewCmd(f))
	c.AddCommand(servicetemplatepublish.NewCmd(f))

	return c
}
