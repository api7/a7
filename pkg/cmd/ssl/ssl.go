package ssl

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	sslcreate "github.com/api7/a7/pkg/cmd/ssl/create"
	ssldelete "github.com/api7/a7/pkg/cmd/ssl/delete"
	sslget "github.com/api7/a7/pkg/cmd/ssl/get"
	ssllist "github.com/api7/a7/pkg/cmd/ssl/list"
	sslupdate "github.com/api7/a7/pkg/cmd/ssl/update"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "ssl",
		Short: "Manage APISIX runtime SSL certificates",
	}

	c.AddCommand(ssllist.NewCmd(f))
	c.AddCommand(sslget.NewCmd(f))
	c.AddCommand(sslcreate.NewCmd(f))
	c.AddCommand(sslupdate.NewCmd(f))
	c.AddCommand(ssldelete.NewCmd(f))

	return c
}
