package config

import (
	"time"

	"github.com/spf13/cobra"
)

// Devbox contains configuration for devbox commands.
type Devbox struct {
	*API
}

// DevboxList contains configuration for listing devboxes.
type DevboxList struct {
	*Devbox
}

// DevboxRegister contains configuration for registering a devbox.
type DevboxRegister struct {
	*Devbox
	Name  string
	Claim bool
}

// AddFlags adds flags for devbox register command.
func (c *DevboxRegister) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Name, "name", "", "name for the devbox (optional, will be generated if not provided)")
	cmd.Flags().BoolVar(&c.Claim, "claim", false, "claim connect session during registration")
}

// DevboxDelete contains configuration for deleting a devbox.
type DevboxDelete struct {
	*Devbox
	Name        string
	Wait        bool
	WaitTimeout time.Duration
}

// AddFlags adds flags for devbox delete command.
func (c *DevboxDelete) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.Wait, "wait", true, "wait for devbox to be deleted")
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 5*time.Minute, "timeout to wait for deletion")
}
