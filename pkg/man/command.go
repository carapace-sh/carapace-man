package man

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/carapace-sh/carapace-spec/pkg/command"
	"gopkg.in/yaml.v3"
)

func cmd(uid *url.URL) (string, error) {
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
