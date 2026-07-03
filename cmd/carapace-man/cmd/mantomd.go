package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-man/pkg/man"
	"github.com/spf13/cobra"
)

var manToMdCmd = &cobra.Command{
	Use:     "man-to-md <command>",
	Short:   "convert man page to markdown",
	GroupID: "main",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		section, err := cmd.Flags().GetString("section")
		if err != nil {
			return err
		}

		output, err := man.ManToMd(args[0], section)
		if err != nil {
			fmt.Fprintf(os.Stderr, "man page not found, falling back to --help: %v\n", err)
			return fallbackHelp(args[0])
		}
		fmt.Print(output)
		return nil
	},
}

func fallbackHelp(name string) error {
	cmd := exec.Command(name, "--help")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("no man page or --help available for %q", name)
	}
	fmt.Print(string(output))
	return nil
}

func init() {
	rootCmd.AddCommand(manToMdCmd)

	manToMdCmd.Flags().String("section", "", "man page section (e.g. 1, 5, 8)")

	carapace.Gen(manToMdCmd).PositionalCompletion(
		carapace.ActionFiles(),
	)
}
