package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carapace-sh/carapace-spec/pkg/command"
	"gopkg.in/yaml.v3"
)

func SplitTo(path string, outputDir string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cmd command.Command
	if err := yaml.Unmarshal(content, &cmd); err != nil {
		return err
	}
	return splitTo(cmd, outputDir, []string{})
}

func splitTo(cmd command.Command, outputDir string, prefix []string) error {
	prefix = append(prefix, strings.SplitN(cmd.Name, " ", 2)[0])

	for _, subcommand := range cmd.Commands {
		if err := splitTo(subcommand, outputDir, prefix); err != nil {
			return err
		}
	}
	cmd.Commands = nil

	filename := filepath.Join(outputDir, strings.Join(prefix, ".")+".yaml")

	if existing, err := os.ReadFile(filename); err == nil {
		var existingCmd command.Command
		if err := yaml.Unmarshal(existing, &existingCmd); err == nil {
			mergeDocumentation(&cmd, existingCmd)
		}
	}

	m, err := yaml.Marshal(cmd)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return err
	}
	fmt.Println(filename)
	return os.WriteFile(filename,
		[]byte("# yaml-language-server: $schema=https://carapace.sh/schemas/command.json\n"+string(m)),
		os.ModePerm)
}

func mergeDocumentation(fresh *command.Command, existing command.Command) {
	if fresh.Documentation.Command == "" && existing.Documentation.Command != "" {
		fresh.Documentation.Command = existing.Documentation.Command
	}
	if fresh.Documentation.Flag == nil && existing.Documentation.Flag != nil {
		fresh.Documentation.Flag = existing.Documentation.Flag
	} else if fresh.Documentation.Flag != nil && existing.Documentation.Flag != nil {
		for k, v := range existing.Documentation.Flag {
			if _, ok := fresh.Documentation.Flag[k]; !ok {
				fresh.Documentation.Flag[k] = v
			}
		}
	}
	if fresh.Documentation.Positional == nil && existing.Documentation.Positional != nil {
		fresh.Documentation.Positional = existing.Documentation.Positional
	}
	if fresh.Documentation.PositionalAny == "" && existing.Documentation.PositionalAny != "" {
		fresh.Documentation.PositionalAny = existing.Documentation.PositionalAny
	}
	if fresh.Documentation.Dash == nil && existing.Documentation.Dash != nil {
		fresh.Documentation.Dash = existing.Documentation.Dash
	}
	if fresh.Documentation.DashAny == "" && existing.Documentation.DashAny != "" {
		fresh.Documentation.DashAny = existing.Documentation.DashAny
	}
	if fresh.Examples == nil && existing.Examples != nil {
		fresh.Examples = existing.Examples
	}
}
