package config

import (
	"github.com/spf13/cobra"
)

type LocalDaemon struct {
	*Local

	// Flags
	RunUnpriveleged bool
	Cluster         string
	Port            uint16
	UID             int
}

func (c *LocalDaemon) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.RunUnpriveleged, "sandbox-manager", false, "run without root priveleges")
	cmd.Flags().StringVar(&c.Cluster, "cluster", "", "signadot cluster name")
	cmd.Flags().Uint16Var(&c.Port, "api-port", 6666, "api service port")
	cmd.Flags().IntVar(&c.UID, "org-uid", -1, "original uid")
}
