package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/carapace-sh/carapace-man/pkg/man"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:     "sync",
	Short:   "clone/pull repo and sync cmd/ docs",
	GroupID: "main",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := man.NewRepo(man.WithProgress(cmd.ErrOrStderr()))
		if err != nil {
			return err
		}
		if err := repo.Sync(); err != nil {
			return err
		}

		cmdSource, _ := cmd.Flags().GetString("cmd-source")
		if cmdSource != "" {
			location, err := man.Location()
			if err != nil {
				return err
			}
			return syncCmdDir(cmdSource, filepath.Join(location, "cmd"), cmd.ErrOrStderr())
		}
		return nil
	},
}

func syncCmdDir(src, dst string, progress io.Writer) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("cmd-source: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		srcDir := filepath.Join(src, entry.Name())
		dstDir := filepath.Join(dst, entry.Name())

		fmt.Fprintf(progress, "syncing cmd/%s\n", entry.Name())
		if err := os.MkdirAll(dstDir, os.ModePerm); err != nil {
			return err
		}

		files, err := os.ReadDir(srcDir)
		if err != nil {
			return err
		}
		for _, f := range files {
			if f.IsDir() || filepath.Ext(f.Name()) != ".yaml" {
				continue
			}
			srcFile := filepath.Join(srcDir, f.Name())
			dstFile := filepath.Join(dstDir, f.Name())
			data, err := os.ReadFile(srcFile)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstFile, data, os.ModePerm); err != nil {
				return err
			}
		}
	}
	return nil
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().String("cmd-source", "", "path to carapace-bin man/cmd/ directory to sync from")
}
