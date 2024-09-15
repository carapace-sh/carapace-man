package cmd

import (
	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-man/cmd/carapace-man/cmd/util"
	"github.com/spf13/cobra"
)

var splitCmd = &cobra.Command{
	Use:     "split <spec>",
	Short:   "split spec",
	GroupID: "main",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return util.Split(args[1])
	},
}

func init() {
	rootCmd.AddCommand(splitCmd)

	carapace.Gen(splitCmd).PositionalCompletion(
		carapace.ActionFiles(),
	)
}
