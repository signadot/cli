package planrunnergroup

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newImage(prg *config.PlanRunnerGroup) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Manage images on a plan runner group",
	}
	cmd.AddCommand(
		newImageList(prg),
		newImagePush(prg),
		newImageDelete(prg),
	)
	return cmd
}
