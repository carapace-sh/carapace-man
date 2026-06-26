package cmd

import (
	"fmt"
	"os"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-man/cmd/carapace-man/cmd/util"
	"github.com/carapace-sh/carapace-spec/pkg/command"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var updateCmd = &cobra.Command{
	Use:     "update <spec> <output-dir>",
	Short:   "update cmd docs for a completer",
	GroupID: "main",
	Long: `Update man/cmd/ docs from a fresh spec file.

Splits the spec into per-subcommand files in the output directory,
preserving existing documentation entries, and reports what changed
or needs documentation.

This is the recommended one-step way to update man/cmd/ docs:

  carapace <completer> spec | carapace-man update - man/cmd/<completer>

Or with a pre-generated spec file:

  carapace-man update /tmp/<completer>-spec.yaml man/cmd/<completer>

Steps performed:
  1. spec-diff: report changes and missing docs
  2. split-to: write updated spec files (preserving documentation)`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		specPath := args[0]
		outputDir := args[1]
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Handle stdin
		if specPath == "-" {
			tmpFile, err := os.CreateTemp("", "carapace-man-update-*.yaml")
			if err != nil {
				return err
			}
			tmpPath := tmpFile.Name()
			defer os.Remove(tmpPath)

			if _, err := tmpFile.ReadFrom(os.Stdin); err != nil {
				tmpFile.Close()
				return err
			}
			tmpFile.Close()
			specPath = tmpPath
		}

		// Validate spec file
		content, err := os.ReadFile(specPath)
		if err != nil {
			return err
		}
		var cmdSpec command.Command
		if err := yaml.Unmarshal(content, &cmdSpec); err != nil {
			return fmt.Errorf("invalid spec file: %w", err)
		}

		// Show diff before making changes
		fmt.Println("Current diff:")
		result, err := util.SpecDiff(specPath, outputDir)
		if err != nil {
			return fmt.Errorf("failed to compute diff: %w", err)
		}
		util.PrintDiff(result)
		fmt.Println()

		if dryRun {
			fmt.Println("Dry run — no files written.")
			return nil
		}

		// Split spec to directory (preserves existing documentation)
		if err := util.SplitTo(specPath, outputDir); err != nil {
			return fmt.Errorf("failed to split spec: %w", err)
		}

		fmt.Println("\nUpdate complete.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().Bool("dry-run", false, "show what would change without writing files")

	carapace.Gen(updateCmd).PositionalCompletion(
		carapace.ActionFiles(".yaml"),
		carapace.ActionDirectories(),
	)
}
