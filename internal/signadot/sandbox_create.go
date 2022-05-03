package signadot

import (
	"github.com/spf13/cobra"
)

type sandboxCreateCmd struct {
	*cobra.Command

	// Parent commands
	root    *RootCmd
	sandbox *sandboxCmd

	// Flags
	filename string
}

func addSandboxCreateCmd(sandbox *sandboxCmd) {
	c := &sandboxCreateCmd{
		root:    sandbox.root,
		sandbox: sandbox,
	}
	c.Command = &cobra.Command{
		Use:   "create -f FILENAME",
		Short: "Create sandbox",
		Args:  cobra.NoArgs,
		RunE:  c.run,
	}

	c.Flags().StringVarP(&c.filename, "filename", "f", "", "YAML or JSON file containing the sandbox creation request")
	c.MarkFlagRequired("filename")

	sandbox.AddCommand(c.Command)
}

func (c *sandboxCreateCmd) run(cmd *cobra.Command, args []string) error {
	// TODO: Implement sandbox create.

	return nil
}
