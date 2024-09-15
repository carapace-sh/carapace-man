package cmd

import (
	"fmt"
	"net/url"
	"os"

	"github.com/carapace-sh/carapace"
	action "github.com/carapace-sh/carapace-man/pkg/actions/man"
	"github.com/carapace-sh/carapace-man/pkg/man"
	spec "github.com/carapace-sh/carapace-spec"
	"github.com/charmbracelet/glamour"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var rootCmd = &cobra.Command{
	Use:  "carapace-man <uid>",
	Args: cobra.ExactArgs(1),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		wrap, err := cmd.Flags().GetInt("wrap")
		if err != nil {
			return err
		}
		return printUid(args[0], cmd.Flag("style").Value.String(), wrap)
	},
}

func printUid(s, style string, width int) error {
	uid, err := url.Parse(s)
	if err != nil {
		return err
	}

	opts := make([]glamour.TermRendererOption, 0)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		if width == 0 {
			width, _, err = term.GetSize(int(os.Stdout.Fd()))
			if err != nil {
				return err
			}
		}

		opts = append(opts,
			glamour.WithStylePath(style),
			glamour.WithWordWrap(width),
		)
	}
	description, err := man.Describe(uid, opts...)
	if err != nil {
		return err
	}

	fmt.Println(description)
	return nil
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.SetHelpCommand(&cobra.Command{Use: "_carapace_help", Hidden: true, Deprecated: "fake help command to prevent default"})
	rootCmd.AddGroup(
		&cobra.Group{ID: "main", Title: "main commands"},
	)

	rootCmd.Flags().String("style", "carapace", "style name or json path")
	rootCmd.Flags().Int("wrap", 0, "word wrap")

	carapace.Gen(rootCmd).FlagCompletion(carapace.ActionMap{
		"style": carapace.Batch(
			spec.ActionMacro("$carapace.tools.glow.Styles"),
			carapace.ActionValues("carapace").Tag("styles"),
			carapace.ActionFiles(".json"),
		).ToA(),
	})

	carapace.Gen(rootCmd).PositionalCompletion(
		action.ActionUids(),
	)

	carapace.Gen(rootCmd).DashAnyCompletion(
		carapace.ActionPositional(rootCmd),
	)

	spec.AddMacro("Uids", spec.MacroN(action.ActionUids))
	spec.Register(rootCmd)
}
