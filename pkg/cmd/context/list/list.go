package list

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/api7/a7/internal/config"
	cmd "github.com/api7/a7/pkg/cmd"
	"github.com/api7/a7/pkg/cmdutil"
	"github.com/api7/a7/pkg/tableprinter"
)

// Options holds the inputs for context list.
type Options struct {
	Config func() (config.Config, error)
	Output string
}

// NewCmd creates the "context list" command.
func NewCmd(f *cmd.Factory) *cobra.Command {
	opts := &Options{
		Config: f.Config,
	}

	return &cobra.Command{
		Use:     "list",
		Short:   "List all contexts",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			opts.Output, _ = c.Flags().GetString("output")
			return listRun(opts, f)
		},
	}
}

func listRun(opts *Options, f *cmd.Factory) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	contexts := cfg.Contexts()
	current := cfg.CurrentContext()

	if len(contexts) == 0 {
		fmt.Fprintln(f.IOStreams.ErrOut, "No contexts configured. Run 'a7 context create' to add one.")
		return nil
	}

	if opts.Output != "" {
		type contextJSON struct {
			Name          string `json:"name"`
			Server        string `json:"server"`
			GatewayGroup  string `json:"gateway_group,omitempty"`
			TLSSkipVerify bool   `json:"tls_skip_verify,omitempty"`
			CACert        string `json:"ca_cert,omitempty"`
			Current       bool   `json:"current"`
		}
		items := make([]contextJSON, 0, len(contexts))
		for _, ctx := range contexts {
			items = append(items, contextJSON{
				Name:          ctx.Name,
				Server:        ctx.Server,
				GatewayGroup:  ctx.GatewayGroup,
				TLSSkipVerify: ctx.TLSSkipVerify,
				CACert:        ctx.CACert,
				Current:       ctx.Name == current,
			})
		}
		exporter := cmdutil.NewExporter(opts.Output, f.IOStreams.Out)
		return exporter.Write(items)
	}

	tp := tableprinter.New(f.IOStreams.Out)
	tp.SetHeaders("CURRENT", "NAME", "SERVER", "GATEWAY-GROUP")

	for _, ctx := range contexts {
		marker := ""
		if ctx.Name == current {
			marker = "*"
		}
		tp.AddRow(marker, ctx.Name, ctx.Server, ctx.GatewayGroup)
	}

	return tp.Render()
}
