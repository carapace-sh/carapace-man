package cmd

import (
	"fmt"
	"os"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-man/pkg/man"
	"github.com/spf13/cobra"
)

var compileCmd = &cobra.Command{
	Use:     "compile <spec> <output.db>",
	Short:   "compile specs into a bbolt database",
	GroupID: "main",
	Long: `Compile man documentation specs into a bbolt database.

Default mode compiles a carapace-spec command spec:
  carapace-man compile <spec.yaml> <output.db>           # one spec → one DB
  carapace-man compile <spec-dir> <output.db>            # directory of specs → one DB
  carapace-man compile - <output.db>                     # spec from stdin → one DB

Static scheme docs (map[string]string YAML):
  carapace-man compile --static <yaml> <scheme> <host> <output.db>

Compile an entire man/ directory tree:
  carapace-man compile --man-dir <man-dir> <output.db>`,
	Args: cobra.RangeArgs(2, 4),
	RunE: func(cmd *cobra.Command, args []string) error {
		static, _ := cmd.Flags().GetBool("static")
		manDir, _ := cmd.Flags().GetBool("man-dir")

		switch {
		case manDir:
			if len(args) != 2 {
				return fmt.Errorf("--man-dir requires <man-dir> <output.db>")
			}
			return man.CompileManDir(args[0], args[1])

		case static:
			if len(args) != 4 {
				return fmt.Errorf("--static requires <yaml> <scheme> <host> <output.db>")
			}
			return man.CompileStatic(args[0], args[1], args[2], args[3])

		default:
			if len(args) != 2 {
				return fmt.Errorf("compile requires <spec> <output.db>")
			}
			specPath := args[0]
			dbPath := args[1]

			if specPath == "-" {
				return man.CompileSpecReader(os.Stdin, "", dbPath)
			}

			info, err := os.Stat(specPath)
			if err != nil {
				return err
			}
			if info.IsDir() {
				return man.CompileSpecDir(specPath, dbPath)
			}
			return man.CompileSpec(specPath, dbPath)
		}
	},
}

func init() {
	rootCmd.AddCommand(compileCmd)

	compileCmd.Flags().Bool("static", false, "compile a static scheme doc (map[string]string YAML)")
	compileCmd.Flags().Bool("man-dir", false, "compile an entire man/ directory tree")

	carapace.Gen(compileCmd).PositionalCompletion(
		carapace.ActionFiles(".yaml"),
		carapace.ActionFiles(".db"),
	)
}
