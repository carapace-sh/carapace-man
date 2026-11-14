package cmd

import (
	"os"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-man/pkg/man"
	"github.com/spf13/cobra"
)

var compileCmd = &cobra.Command{
	Use:     "compile <file|dir|-> <db>",
	Short:   "compile specs and man docs into a badger database",
	GroupID: "main",
	Long: `Compile man documentation specs into a badger database (zstd compressed).

Compile a single carapace-spec command spec (recognized by its
schema header or command-like content):
  carapace-man compile <spec.yaml> <output.db>           # one spec → one host
  carapace-man compile - <output.db>                     # spec from stdin → one host

Compile an entire man/ directory tree (man/cmd/<host>/*.yaml for the
cmd scheme, <scheme>/<host>/<host>.yaml for generic schemes):
  carapace-man compile <man-dir> <output.db>

The database is rebuilt from scratch unless --amend is given, in which
case processed hosts are replaced and others are preserved.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		amend, _ := cmd.Flags().GetBool("amend")
		var opts []man.CompileOption
		if amend {
			opts = append(opts, man.WithAmend(true))
		}
		source, dbPath := args[0], args[1]

		if source == "-" {
			return man.CompileSpecReader(os.Stdin, "", dbPath, opts...)
		}
		info, err := os.Stat(source)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return man.CompileManDir(source, dbPath, opts...)
		}
		return man.CompileSpec(source, dbPath, opts...)
	},
}

func init() {
	rootCmd.AddCommand(compileCmd)

	compileCmd.Flags().Bool("amend", false, "update an existing database instead of rebuilding it")

	carapace.Gen(compileCmd).PositionalCompletion(
		carapace.ActionFiles(),
		carapace.ActionFiles(".db"),
	)
}
