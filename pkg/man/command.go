package man

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/carapace-sh/carapace/pkg/command"
	"github.com/carapace-sh/carapace/pkg/xdg"
	"go.yaml.in/yaml/v3"
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

	if s, ok, err := cmdDB(uid); err != nil {
		return "", err
	} else if ok {
		return s, nil
	}

	location, err := Location()
	if err != nil {
		return "", err
	}

	subcommand := strings.ReplaceAll(uid.Path, "/", ".")
	path := fmt.Sprintf("%v/%v/%v/%v.yaml", location, uid.Scheme, uid.Host, MangleFilename(uid.Host+subcommand))

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

// CmdFlags returns known documentation flags for a cmd UID (cmd://host/sub/path).
// Results are merged from compiled databases, user specs, and synced YAML specs.
func CmdFlags(host, subPath string) ([]string, error) {
	seen := map[string]bool{}

	if uids, err := UidsFromDBs("cmd", host); err == nil {
		for _, uid := range uids {
			if strings.TrimPrefix(uid.Path, "/") != subPath {
				continue
			}
			if q := uid.Query(); q.Has("flag") {
				if flag := q.Get("flag"); flag != "" {
					seen[flag] = true
				}
			}
		}
	}

	if flags, err := cmdUserFlags(host, subPath); err == nil {
		for _, flag := range flags {
			seen[flag] = true
		}
	}

	if flags, err := cmdRepoFlags(host, subPath); err == nil {
		for _, flag := range flags {
			seen[flag] = true
		}
	}

	flags := make([]string, 0, len(seen))
	for flag := range seen {
		flags = append(flags, flag)
	}
	sort.Strings(flags)
	return flags, nil
}

func cmdUserFlags(host, subPath string) ([]string, error) {
	configDir, err := xdg.UserConfigDir()
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%v/carapace/specs/%v.yaml", configDir, host)

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var specCommand command.Command
	if err := yaml.Unmarshal(content, &specCommand); err != nil {
		return nil, err
	}

	cmd := &specCommand
	if subPath != "" {
		cmd, err = specCommand.Find(strings.Split(subPath, "/"))
		if err != nil {
			return nil, err
		}
	}

	flags := make([]string, 0, len(cmd.Documentation.Flag))
	for flag := range cmd.Documentation.Flag {
		flags = append(flags, flag)
	}
	return flags, nil
}

func cmdRepoFlags(host, subPath string) ([]string, error) {
	location, err := Location()
	if err != nil {
		return nil, err
	}

	subcommand := ""
	if subPath != "" {
		subcommand = "." + strings.ReplaceAll(subPath, "/", ".")
	}
	path := fmt.Sprintf("%v/%v/%v/%v.yaml", location, "cmd", host, MangleFilename(host+subcommand))

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var specCommand command.Command
	if err := yaml.Unmarshal(content, &specCommand); err != nil {
		return nil, err
	}

	flags := make([]string, 0, len(specCommand.Documentation.Flag))
	for flag := range specCommand.Documentation.Flag {
		flags = append(flags, flag)
	}
	return flags, nil
}
