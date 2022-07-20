package cmd

import (
	"errors"
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/spf13/cobra"
)

var updateFrom string

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update rum",
	Run: func(cmd *cobra.Command, args []string) {
		err := errors.New(fmt.Sprintf("invalid `-from`: %s", updateFrom))
		if updateFrom == "qingcloud" {
			err = utils.CheckUpdateQingCloud(utils.ReleaseVersion, "quorum")
		} else if updateFrom == "github" {
			err = utils.CheckUpdate(utils.ReleaseVersion, "quorum")
		} else {
			err = errors.New("only support github or qingcloud")
		}
		if err != nil {
			logger.Fatalf("Failed to do self-update: %s", err.Error())
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	flags := updateCmd.Flags()
	flags.SortFlags = false
	flags.StringVarP(&updateFrom, "from", "f", "github", "update from: github/qingcloud")
}
