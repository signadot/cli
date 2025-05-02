package config

import "github.com/spf13/cobra"

type Auth struct {
	*API
}

type AuthLogin struct {
	*Auth

	WithAPIKey string
}

func (c *AuthLogin) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.WithAPIKey, "with-api-key", "", "log in using the provided API key.")
}

type AuthStatus struct {
	*Auth
}

type AuthLogout struct {
	*Auth
}
