package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carapace-sh/carapace-man/cmd/carapace-man/cmd/util/ollama"
	spec "github.com/carapace-sh/carapace-spec"
	"github.com/carapace-sh/carapace-spec/pkg/command"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

func Enrich(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var command command.Command
	if err := yaml.Unmarshal(content, &command); err != nil {
		return err
	}

	trimmed := filepath.Base(path)
	trimmed = strings.TrimSuffix(trimmed, filepath.Ext(trimmed))
	prefix := []string{}
	if strings.Contains(trimmed, ".") {
		trimmed = strings.TrimSuffix(trimmed, filepath.Ext(trimmed))
		prefix = strings.Split(trimmed, ".")
	}
	enriched, err := enrich(command, prefix...)
	if err != nil {
		return err
	}

	return save(*enriched, prefix...)
}

func enrich(command command.Command, prefix ...string) (*command.Command, error) {
	cobraCommand, err := spec.Command(command).ToCobraE()
	if err != nil {
		return nil, err
	}

	name := strings.Join(append(prefix, cobraCommand.Name()), " ")
	client, err := ollama.NewClient("mistral")
	if err != nil {
		return nil, err
	}

	if command.Documentation.Command == "" {
		println(fmt.Sprintf("requesting %#v", name))
		content, err := client.ExplainCommand(name)
		if err != nil {
			return nil, err
		}
		command.Documentation.Command = "[AI] " + strings.TrimSpace(content)
	}

	if command.Documentation.Flag == nil {
		command.Documentation.Flag = make(map[string]string)
	}
	cobraCommand.Flags().VisitAll(func(f *pflag.Flag) {
		if command.Documentation.Flag[f.Name] == "" {
			var flag string
			switch f.Mode {
			case pflag.ShorthandOnly:
				flag = "-" + f.Shorthand
			case pflag.NameAsShorthand:
				flag = "-" + f.Name
			default:
				flag = "--" + f.Name
			}
			println(fmt.Sprintf("requesting %#v", name+" "+flag))
			content, err := client.ExplainFlag(name, flag)
			// TODO handle error
			if err == nil {
				command.Documentation.Flag[f.Name] = "[AI] " + strings.TrimSpace(content)
			}
		}
	})
	return &command, nil
}

// TODO re-enable
func overwrite(filename string, command command.Command) error {
	m, err := yaml.Marshal(command)
	if err != nil {
		return err
	}
	println(filename)

	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(filename,
		[]byte("# yaml-language-server: $schema=https://carapace.sh/schemas/command.json\n"+string(m)),
		os.ModePerm)
}
