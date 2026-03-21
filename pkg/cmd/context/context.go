package context

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	contextcreate "github.com/api7/a7/pkg/cmd/context/create"
	contextcurrent "github.com/api7/a7/pkg/cmd/context/current"
	contextdelete "github.com/api7/a7/pkg/cmd/context/delete"
	contextlist "github.com/api7/a7/pkg/cmd/context/list"
	contextuse "github.com/api7/a7/pkg/cmd/context/use"
)

// NewCmd creates the context parent command.
func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "context",
		Short: "Manage connection contexts for API7 EE instances",
		Long:  "Create, switch, list, and delete named connection profiles.\nEach context stores a server URL, API token, and optional gateway group.",
	}

	c.AddCommand(contextcreate.NewCmd(f))
	c.AddCommand(contextuse.NewCmd(f))
	c.AddCommand(contextlist.NewCmd(f))
	c.AddCommand(contextdelete.NewCmd(f))
	c.AddCommand(contextcurrent.NewCmd(f))

	return c
}
