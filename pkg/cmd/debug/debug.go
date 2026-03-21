package debug

import (
	"github.com/spf13/cobra"

	"github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmd/debug/logs"
	"github.com/api7/a7/pkg/cmd/debug/trace"
)

func NewCmd(f *cmd.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug and diagnose API7 EE",
	}

	cmd.AddCommand(logs.NewCmdLogs(f))
	cmd.AddCommand(trace.NewCmdTrace(f))

	return cmd
}
