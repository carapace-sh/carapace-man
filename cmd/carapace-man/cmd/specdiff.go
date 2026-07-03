package cmd

import (
	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-man/cmd/carapace-man/cmd/util"
	"github.com/spf13/cobra"
)

var specDiffCmd = &cobra.Command{
	Use:     "spec-diff <spec> <existing-dir>",
	Short:   "diff fresh spec against existing cmd docs",
	GroupID: "main",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := util.SpecDiff(args[0], args[1])
		if err != nil {
			return err
		}
		util.PrintDiff(result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(specDiffCmd)

	carapace.Gen(specDiffCmd).PositionalCompletion(
		carapace.ActionFiles(),
		carapace.ActionDirectories(),
	)
}
