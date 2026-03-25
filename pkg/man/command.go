package man

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/carapace-sh/carapace-spec/pkg/command"
	"github.com/carapace-sh/carapace/pkg/xdg"
	"gopkg.in/yaml.v3"
)

func cmd_user(uid *url.URL) (string, error) {
	configDir, err := xdg.UserConfigDir()
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("%v/carapace/specs/%v.yaml", configDir, uid.Host)

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var specCommand command.Command
	if err := yaml.Unmarshal(content, &specCommand); err != nil {
		return "", err
	}

	command := &specCommand
	if path := strings.TrimPrefix(uid.Path, "/"); path != "" {
		command, err = specCommand.Find(strings.Split(strings.TrimPrefix(uid.Path, "/"), "/"))
		if err != nil {
			return "", err
		}
	}

	if q := uid.Query(); q.Has("flag") {
		return command.Documentation.Flag[q.Get("flag")], nil
	}
	return command.Documentation.Command, nil
}

func cmd(uid *url.URL) (string, error) {
	if s, err := cmd_user(uid); err == nil {
		return s, nil
	}

	location, err := Location()
	if err != nil {
		return "", err
	}

	subcommand := strings.ReplaceAll(uid.Path, "/", ".")
	path := fmt.Sprintf("%v/%v/%v/%v%v.yaml", location, uid.Scheme, uid.Host, uid.Host, subcommand)

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var specCommand command.Command
	if err := yaml.Unmarshal(content, &specCommand); err != nil {
		return "", err
	}

	if q := uid.Query(); q.Has("flag") {
		return specCommand.Documentation.Flag[q.Get("flag")], nil
	}

	return specCommand.Documentation.Command, nil

}
