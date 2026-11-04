package cmd

import (
	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-man/pkg/mdbook"
	"github.com/spf13/cobra"
)

var mdbookCmd = &cobra.Command{
	Use:     "mdbook <archive-dir> <book-dir>",
	Short:   "build an mdbook from an archived man page tree",
	GroupID: "main",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return mdbook.Generate(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(mdbookCmd)

	carapace.Gen(mdbookCmd).PositionalCompletion(
		carapace.ActionDirectories(),
		carapace.ActionDirectories(),
	)
}
