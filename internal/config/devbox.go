package config

import (
	"github.com/spf13/cobra"
)

// Devbox contains configuration for devbox commands.
type Devbox struct {
	*API
}

// DevboxList contains configuration for listing devboxes.
type DevboxList struct {
	*Devbox

	// Flags
	ShowAll bool
}

// AddFlags adds flags for devbox list command.
func (c *DevboxList) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.ShowAll, "all", false, "list all devboxes")
}

// DevboxRegister contains configuration for registering a devbox.
type DevboxRegister struct {
	*Devbox
	Name string
}

// AddFlags adds flags for devbox register command.
func (c *DevboxRegister) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Name, "name", "", "name for the devbox (optional, will be generated if not provided)")
}

// DevboxDelete contains configuration for deleting a devbox.
type DevboxDelete struct {
	*Devbox
	ID string
}

// AddFlags adds flags for devbox delete command.
func (c *DevboxDelete) AddFlags(cmd *cobra.Command) {
	// No flags currently
}
