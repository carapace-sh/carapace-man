package cmd

import (
	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-man/cmd/carapace-man/cmd/util"
	"github.com/spf13/cobra"
)

var enrichCmd = &cobra.Command{
	Use:     "enrich <spec>",
	Short:   "enrich spec",
	GroupID: "main",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return util.Enrich(args[1])
	},
}

func init() {
	rootCmd.AddCommand(enrichCmd)

	carapace.Gen(enrichCmd).PositionalCompletion(
		carapace.ActionFiles(),
	)
}
