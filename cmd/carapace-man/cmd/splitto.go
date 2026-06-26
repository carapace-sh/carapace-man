package cmd

import (
	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-man/cmd/carapace-man/cmd/util"
	"github.com/spf13/cobra"
)

var splitToCmd = &cobra.Command{
	Use:     "split-to <spec> <output-dir>",
	Short:   "split spec to directory",
	GroupID: "main",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return util.SplitTo(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(splitToCmd)

	carapace.Gen(splitToCmd).PositionalCompletion(
		carapace.ActionFiles(),
		carapace.ActionDirectories(),
	)
}
