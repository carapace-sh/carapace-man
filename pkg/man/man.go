package man

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"charm.land/glamour/v2"
	"github.com/carapace-sh/carapace/pkg/xdg"
	"github.com/mattn/go-isatty"
	"go.yaml.in/yaml/v3"
	"golang.org/x/term"
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
	seen := map[string]bool{}

	location, err := Location()
	if err != nil {
		return nil, err
	}
	if entries, err := os.ReadDir(location); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				seen[entry.Name()] = true
			}
		}
	}

	dbSchemes, err := SchemesFromDBs()
	if err != nil {
		return nil, err
	}
	for _, scheme := range dbSchemes {
		seen[scheme] = true
	}

	schemes := make([]string, 0, len(seen))
	for scheme := range seen {
		schemes = append(schemes, scheme)
	}
	sort.Strings(schemes)
	return schemes, nil
}

func Hosts(scheme string) ([]string, error) {
	seen := map[string]bool{}

	location, err := Location()
	if err != nil {
		return nil, err
	}
	if entries, err := os.ReadDir(fmt.Sprintf("%v/%v", location, scheme)); err == nil { // TODO prevent `..` and similar
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				seen[entry.Name()] = true
			}
		}
	}

	dbHosts, err := HostsFromDBs(scheme)
	if err != nil {
		return nil, err
	}
	for _, host := range dbHosts {
		seen[host] = true
	}

	hosts := make([]string, 0, len(seen))
	for host := range seen {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	return hosts, nil
}

func Uids(scheme, host string) ([]*url.URL, error) {
	seen := map[string]bool{}
	uids := make([]*url.URL, 0)

	location, err := Location()
	if err != nil {
		return nil, err
	}
	if entries, err := os.ReadDir(fmt.Sprintf("%v/%v/%v", location, scheme, host)); err == nil { // TODO prevent `..` and similar
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
			if !seen[uid.String()] {
				seen[uid.String()] = true
				uids = append(uids, uid)
			}
		}
	}

	dbUids, err := UidsFromDBs(scheme, host)
	if err != nil {
		return nil, err
	}
	for _, uid := range dbUids {
		if uid.RawQuery != "" {
			continue
		}
		if !seen[uid.String()] {
			seen[uid.String()] = true
			uids = append(uids, uid)
		}
	}

	sort.Slice(uids, func(i, j int) bool {
		return uids[i].String() < uids[j].String()
	})
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
	return description, err
}

func Style(s string, opts ...glamour.TermRendererOption) (string, error) {
	r, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		return "", err
	}
	return r.Render(s)
}

// RenderOptions builds glamour renderer options for styling terminal output.
// Returns nil opts if output is not a TTY or raw mode is requested,
// so callers can skip styling entirely.
func RenderOptions(style string, width int, raw bool) ([]glamour.TermRendererOption, error) {
	if raw || !isatty.IsTerminal(os.Stdout.Fd()) {
		return nil, nil
	}
	if width == 0 {
		var err error
		width, _, err = term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			return nil, err
		}
	}
	return []glamour.TermRendererOption{
		glamour.WithStylePath(style),
		glamour.WithWordWrap(width),
	}, nil
}

func descibe(uid *url.URL) (string, error) {
	if s, ok, err := describeDB(uid); err != nil {
		return "", err
	} else if ok {
		return s, nil
	}

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

	if suffix, ok := strings.CutPrefix(description, "./"); ok { // description is a path
		path := fmt.Sprintf("%v/%v/%v/%v", location, uid.Scheme, uid.Host, suffix) // TODO ensure path is within dir

		content, err = os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	return description, nil // TODO return error for unknown?
}
