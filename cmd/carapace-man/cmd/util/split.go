package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carapace-sh/carapace-spec/pkg/command"
	"gopkg.in/yaml.v3"
)

func Split(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var command command.Command
	if err := yaml.Unmarshal(content, &command); err != nil {
		return err
	}
	if err := split(command); err != nil {
		return err
	}
	return nil
}

func split(command command.Command, prefix ...string) error {
	prefix = append(prefix, strings.SplitN(command.Name, " ", 2)[0])
	for _, subcommand := range command.Commands {
		if err := split(subcommand, prefix...); err != nil {
			return err
		}
	}
	command.Commands = nil
	m, err := yaml.Marshal(command)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%v/carapace-man/%v/%v.yaml", os.TempDir(), prefix[0], strings.Join(prefix, "."))
	println(filename)

	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(filename, m, os.ModePerm)
}

func save(command command.Command, prefix ...string) error {
	prefix = append(prefix, strings.SplitN(command.Name, " ", 2)[0])
	m, err := yaml.Marshal(command)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%v/carapace-man/%v/%v.yaml", os.TempDir(), prefix[0], strings.Join(prefix, "."))
	println(filename)

	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(filename,
		[]byte("# yaml-language-server: $schema=https://carapace.sh/schemas/command.json\n"+string(m)),
		os.ModePerm)
}
