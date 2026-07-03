package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carapace-sh/carapace-spec/pkg/command"
	"gopkg.in/yaml.v3"
)

type DiffResult struct {
	NewSubcommands     []string
	RemovedSubcommands []string
	ChangedFlags       []string
	MissingDocCommand  []string
	AIPrefixedEntries  []string
}

func SpecDiff(path string, existingDir string) (*DiffResult, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cmd command.Command
	if err := yaml.Unmarshal(content, &cmd); err != nil {
		return nil, err
	}

	result := &DiffResult{}
	newSpecs := collectSpecs(cmd, []string{})
	existingSpecs := collectExistingSpecs(existingDir)

	for name := range newSpecs {
		if _, ok := existingSpecs[name]; !ok {
			result.NewSubcommands = append(result.NewSubcommands, name)
		}
	}

	for name := range existingSpecs {
		if _, ok := newSpecs[name]; !ok {
			result.RemovedSubcommands = append(result.RemovedSubcommands, name)
		}
	}

	for name, newCmd := range newSpecs {
		if existingCmd, ok := existingSpecs[name]; ok {
			diffFlags(result, name, newCmd, existingCmd)
		}
		if newCmd.Documentation.Command == "" {
			result.MissingDocCommand = append(result.MissingDocCommand, name)
		}
	}

	for name, existingCmd := range existingSpecs {
		if strings.HasPrefix(existingCmd.Documentation.Command, "[AI] ") {
			result.AIPrefixedEntries = append(result.AIPrefixedEntries, name+"/command")
		}
		for k, v := range existingCmd.Documentation.Flag {
			if strings.HasPrefix(v, "[AI] ") {
				result.AIPrefixedEntries = append(result.AIPrefixedEntries, name+"/flag/"+k)
			}
		}
	}

	return result, nil
}

func collectSpecs(cmd command.Command, prefix []string) map[string]command.Command {
	result := make(map[string]command.Command)
	collectSpecsInto(cmd, prefix, result)
	return result
}

func collectSpecsInto(cmd command.Command, prefix []string, result map[string]command.Command) {
	prefix = append(prefix, strings.SplitN(cmd.Name, " ", 2)[0])
	key := strings.Join(prefix, ".")
	subcommands := cmd.Commands
	cmd.Commands = nil
	result[key] = cmd

	for _, sub := range subcommands {
		collectSpecsInto(sub, prefix, result)
	}
}

func collectExistingSpecs(dir string) map[string]command.Command {
	result := make(map[string]command.Command)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return result
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cmd command.Command
		if err := yaml.Unmarshal(content, &cmd); err != nil {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		result[name] = cmd
	}
	return result
}

func diffFlags(result *DiffResult, name string, newCmd, existingCmd command.Command) {
	for flag := range newCmd.Flags {
		newDesc := newCmd.Flags[flag].Description
		if existingFlag, ok := existingCmd.Flags[flag]; ok {
			if existingFlag.Description != newDesc {
				result.ChangedFlags = append(result.ChangedFlags, name+"/"+flag)
			}
		} else {
			result.NewSubcommands = append(result.NewSubcommands, name+"/"+flag)
		}
	}
}

func PrintDiff(result *DiffResult) {
	if len(result.NewSubcommands) > 0 {
		fmt.Println("NEW SUBCOMMANDS/FLAGS:")
		for _, s := range result.NewSubcommands {
			fmt.Printf("  + %s\n", s)
		}
	}
	if len(result.RemovedSubcommands) > 0 {
		fmt.Println("REMOVED SUBCOMMANDS:")
		for _, s := range result.RemovedSubcommands {
			fmt.Printf("  - %s\n", s)
		}
	}
	if len(result.ChangedFlags) > 0 {
		fmt.Println("CHANGED FLAGS:")
		for _, s := range result.ChangedFlags {
			fmt.Printf("  ~ %s\n", s)
		}
	}
	if len(result.MissingDocCommand) > 0 {
		fmt.Println("MISSING DOCUMENTATION.COMMAND:")
		for _, s := range result.MissingDocCommand {
			fmt.Printf("  ? %s\n", s)
		}
	}
	if len(result.AIPrefixedEntries) > 0 {
		fmt.Println("[AI] PREFIXED ENTRIES:")
		for _, s := range result.AIPrefixedEntries {
			fmt.Printf("  ! %s\n", s)
		}
	}

	if len(result.NewSubcommands) == 0 && len(result.RemovedSubcommands) == 0 &&
		len(result.ChangedFlags) == 0 && len(result.MissingDocCommand) == 0 &&
		len(result.AIPrefixedEntries) == 0 {
		fmt.Println("No differences found.")
	}
}