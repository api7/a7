package config

import (
	"github.com/spf13/cobra"

	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/config/diff"
	"github.com/api7/a7/pkg/cmd/config/dump"
	"github.com/api7/a7/pkg/cmd/config/sync"
	"github.com/api7/a7/pkg/cmd/config/validate"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "config <command>",
		Short: "Manage declarative API7 EE configuration",
	}

	c.AddCommand(dump.NewCmdDump(f))
	c.AddCommand(diff.NewCmdDiff(f))
	c.AddCommand(sync.NewCmdSync(f))
	c.AddCommand(validate.NewCmdValidate(f))

	return c
}
