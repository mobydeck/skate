package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"skate/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print skate version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("skate %s\n", version.Version)
	},
}
