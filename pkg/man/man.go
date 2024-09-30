package man

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/carapace-sh/carapace/pkg/xdg"
	"github.com/charmbracelet/glamour"
	"gopkg.in/yaml.v3"
)

func Location() (string, error) {
	// TODO override location with environment variable
	configDir, err := xdg.UserConfigDir()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v/carapace/man", configDir), nil
}

func Schemes() ([]string, error) {
	location, err := Location()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(location)
	if err != nil {
		return nil, err
	}

	schemes := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			schemes = append(schemes, entry.Name())
		}
	}
	return schemes, nil
}

func Hosts(scheme string) ([]string, error) {
	location, err := Location()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fmt.Sprintf("%v/%v", location, scheme)) // TODO prevent `..` and similar
	if err != nil {
		return nil, err
	}

	schemes := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			schemes = append(schemes, entry.Name())
		}
	}
	return schemes, nil
}

func Uids(scheme, host string) ([]*url.URL, error) {
	location, err := Location()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fmt.Sprintf("%v/%v/%v", location, scheme, host)) // TODO prevent `..` and similar
	if err != nil {
		return nil, err
	}

	uids := make([]*url.URL, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") || !strings.HasPrefix(entry.Name(), host+".") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		name = strings.ReplaceAll(name, ".", "/")
		name = fmt.Sprintf("%v://%v", scheme, name)

		uid, err := url.Parse(name)
		if err != nil {
			return nil, err
		}
		uids = append(uids, uid)
	}
	return uids, nil
}

func Describe(uid *url.URL, opts ...glamour.TermRendererOption) (description string, err error) {
	switch uid.Scheme {
	case "cmd":
		description, err = cmd(uid)
	case "man":
		description, err = manpage(uid)
	default:
		description, err = descibe(uid)
	}
	if err == nil && len(opts) > 0 {
		description, err = Style(description, opts...)
	}
	return strings.Trim(description, "\n"), err
}

func Style(s string, opts ...glamour.TermRendererOption) (string, error) {
	r, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		return "", err
	}
	return r.Render(s)
}

func descibe(uid *url.URL) (string, error) {
	location, err := Location()
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(fmt.Sprintf("%v/%v/%v/%v.yaml", location, uid.Scheme, uid.Host, uid.Host))
	if err != nil {
		return "", err
	}

	var m map[string]string
	if err := yaml.Unmarshal(content, &m); err != nil {
		return "", err
	}

	description, ok := m[strings.TrimPrefix(uid.Path, "/")]
	if !ok {
		return "", nil
	}

	if strings.HasPrefix(description, "./") { // description is a path
		path := fmt.Sprintf("%v/%v/%v/%v", location, uid.Scheme, uid.Host, strings.TrimPrefix(description, "./")) // TODO ensure path is within dir

		content, err = os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	return m[strings.TrimPrefix(uid.Path, "/")], nil // TODO return error for unknown?
}
