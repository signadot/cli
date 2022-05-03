package signadot

import "github.com/spf13/cobra"

type sandboxCmd struct {
	*cobra.Command

	// Parent commands
	root *RootCmd
}

func addSandboxCmd(root *RootCmd) {
	c := &sandboxCmd{root: root}
	c.Command = &cobra.Command{
		Use:   "sandbox",
		Short: "Inspect and manipulate sandboxes",
	}

	// Subcommands
	addSandboxGetCmd(c)
	addSandboxListCmd(c)
	addSandboxCreateCmd(c)
	addSandboxDeleteCmd(c)

	root.AddCommand(c.Command)
}
