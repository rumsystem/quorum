package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s - %s", ReleaseVersion, GitCommit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
