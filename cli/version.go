package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	Run: func(cmd *cobra.Command, args []string) {
		runVersion()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion() {
	fmt.Println(Version)
}
