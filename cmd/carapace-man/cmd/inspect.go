package cmd

import (
	"fmt"
	"net/url"
	"os"
	"sort"

	"github.com/carapace-sh/carapace"
	action "github.com/carapace-sh/carapace-man/pkg/actions/man"
	"github.com/carapace-sh/carapace-man/pkg/man"
	spec "github.com/carapace-sh/carapace-spec"
	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:     "inspect <db> [uid]",
	Short:   "inspect contents of a compiled bbolt database",
	GroupID: "main",
	Long: `Inspect a compiled bbolt documentation database.

Without a uid, lists all UIDs in the database:

  carapace-man inspect carapace.db

With a uid, renders the documentation for that uid:

  carapace-man inspect carapace.db cmd://git/commit
  carapace-man inspect carapace.db git://color/normal`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := args[0]
		if _, err := os.Stat(dbPath); err != nil {
			return fmt.Errorf("db file: %w", err)
		}

		if len(args) < 2 {
			return listUids(dbPath)
		}

		style, err := cmd.Flags().GetString("style")
		if err != nil {
			return err
		}
		raw, err := cmd.Flags().GetBool("raw")
		if err != nil {
			return err
		}
		wrap, err := cmd.Flags().GetInt("wrap")
		if err != nil {
			return err
		}
		return describeUid(dbPath, args[1], style, raw, wrap)
	},
}

func listUids(dbPath string) error {
	schemes, err := man.InspectSchemes(dbPath)
	if err != nil {
		return err
	}

	var uids []*url.URL
	for _, scheme := range schemes {
		hosts, err := man.InspectHosts(dbPath, scheme)
		if err != nil {
			return err
		}
		for _, host := range hosts {
			hostUids, err := man.InspectUids(dbPath, scheme, host)
			if err != nil {
				return err
			}
			uids = append(uids, hostUids...)
		}
	}

	sort.Slice(uids, func(i, j int) bool {
		return uids[i].String() < uids[j].String()
	})

	for _, uid := range uids {
		fmt.Println(uid.String())
	}
	return nil
}

func describeUid(dbPath, uidStr, style string, raw bool, wrap int) error {
	uid, err := url.Parse(uidStr)
	if err != nil {
		return err
	}

	description, found, err := man.InspectDescribe(dbPath, uid)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no documentation for %q in %s", uidStr, dbPath)
	}

	opts, err := man.RenderOptions(style, wrap, raw)
	if err != nil {
		return err
	}
	if len(opts) > 0 {
		description, err = man.Style(description, opts...)
		if err != nil {
			return err
		}
	}

	fmt.Println(description)
	return nil
}

func init() {
	rootCmd.AddCommand(inspectCmd)

	inspectCmd.Flags().Bool("raw", false, "raw output")
	inspectCmd.Flags().String("style", "carapace", "style name or json path")
	inspectCmd.Flags().Int("wrap", 0, "word wrap")

	carapace.Gen(inspectCmd).FlagCompletion(carapace.ActionMap{
		"style": carapace.Batch(
			spec.ActionMacro("$carapace.tools.glow.Styles"),
			carapace.ActionValues("carapace").Tag("styles"),
			carapace.ActionFiles(".json"),
		).ToA(),
	})

	carapace.Gen(inspectCmd).PositionalCompletion(
		carapace.ActionFiles(".db"),
		carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			if len(c.Args) == 0 {
				return carapace.ActionMessage("db argument required")
			}
			return action.ActionInspectUids(c.Args[0])
		}),
	)
}
