package signadot

import (
	"errors"

	"github.com/spf13/cobra"
)

type sandboxDeleteCmd struct {
	*cobra.Command

	// Parent commands
	root    *RootCmd
	sandbox *sandboxCmd

	// Flags
	filename string
}

func addSandboxDeleteCmd(sandbox *sandboxCmd) {
	c := &sandboxDeleteCmd{
		root:    sandbox.root,
		sandbox: sandbox,
	}
	c.Command = &cobra.Command{
		Use:   "delete { -f FILENAME | NAME }",
		Short: "Delete sandbox",
		Args:  cobra.MaximumNArgs(1),
		RunE:  c.run,
	}

	c.Flags().StringVarP(&c.filename, "filename", "f", "", "optional YAML or JSON file containing the original sandbox creation request")

	sandbox.AddCommand(c.Command)
}

func (c *sandboxDeleteCmd) run(cmd *cobra.Command, args []string) error {
	if c.filename == "" && len(args) == 0 {
		return errors.New("must specify either filename or sandbox name")
	}
	if c.filename != "" && len(args) > 0 {
		return errors.New("can't specify both filename and sandbox name")
	}

	// TODO: Implement sandbox delete.

	return nil
}
