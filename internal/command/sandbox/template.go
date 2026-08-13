package sandbox

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/render"
	"github.com/spf13/cobra"
)

func newTemplate(sandbox *config.Sandbox) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Inspect the built-in sandbox template",
	}
	cmd.AddCommand(newTemplateShow(sandbox))
	return cmd
}

func newTemplateShow(sandbox *config.Sandbox) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Print the built-in sandbox template",
		Long: `Print the built-in sandbox template.

This is the template that renders a values document passed to
"signadot sandbox apply -f". Nothing about it is privileged: save it, edit it,
and pass it to -f to take over the structure yourself.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return templateShow(cmd.OutOrStdout())
		},
	}
	return cmd
}

func templateShow(out io.Writer) error {
	_, err := fmt.Fprint(out, string(render.BuiltinTemplate()))
	return err
}
