package cmd

import (
	"github.com/carapace-sh/carapace-man/pkg/man"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:     "sync",
	Short:   "clone/pull repo",
	GroupID: "main",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := man.NewRepo(man.WithProgress(cmd.ErrOrStderr()))
		if err != nil {
			return err
		}
		return repo.Sync()
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
