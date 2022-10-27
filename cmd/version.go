package cmd

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	Run: func(cmd *cobra.Command, args []string) {
		version := fmt.Sprintf("%s - %s", utils.ReleaseVersion, utils.GitCommit)
		fmt.Println(version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
