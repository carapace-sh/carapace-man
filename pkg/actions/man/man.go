package man

import (
	"fmt"
	"os"
	"strings"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-man/pkg/man"
	spec "github.com/carapace-sh/carapace-spec"
	"gopkg.in/yaml.v3"
)

func ActionUids() carapace.Action {
	return carapace.ActionMultiPartsN("://", 2, func(c carapace.Context) carapace.Action {
		switch len(c.Parts) {
		case 0:
			schemes, err := man.Schemes()
			if err != nil {
				return carapace.ActionMessage(err.Error())
			}
			for index := range schemes {
				schemes[index] += "://"
			}
			return carapace.ActionValues(append(schemes, "man://")...)
		default:
			scheme := c.Parts[0]
			switch scheme {
			case "cmd":
				return actionCmds().NoSpace()
			case "man":
				return carapace.ActionMultiPartsN("/", 2, func(c carapace.Context) carapace.Action {
					switch len(c.Parts) {
					case 0:
						return spec.ActionMacro("$carapace.tools.man.Pages")
					default:
						return spec.ActionMacro("$carapace.tools.man.Sections(%v)", c.Parts[0])
					}
				})
			default:
				return actionOther(scheme)
			}
		}
	}).Tag("uids") // TODO nospace should be implicit
}

func actionCmds() carapace.Action {
	return carapace.ActionMultiPartsN("?", 2, func(c carapace.Context) carapace.Action {
		switch len(c.Parts) {
		case 0:
			return carapace.ActionMultiPartsN("/", 2, func(c carapace.Context) carapace.Action {
				switch len(c.Parts) {
				case 0:
					hosts, err := man.Hosts("cmd")
					if err != nil {
						return carapace.ActionMessage(err.Error())
					}
					return carapace.ActionValues(hosts...)
				default:
					uids, err := man.Uids("cmd", c.Parts[0])
					if err != nil {
						return carapace.ActionMessage(err.Error())
					}

					vals := make([]string, 0, len(uids))
					for _, uid := range uids {
						vals = append(vals, strings.TrimPrefix(uid.Path, "/"))
					}
					return carapace.ActionValues(vals...).MultiParts("/")
				}
			})
		default:
			return carapace.ActionMultiPartsN("=", 2, func(c carapace.Context) carapace.Action {
				switch len(c.Parts) {
				case 0:
					return carapace.ActionValues("flag").Suffix("=")
				default:
					switch c.Parts[0] {
					case "flag":
						return carapace.ActionValues("TODO") // TODO parse spec and return flags
					default:
						return carapace.ActionValues()
					}
				}
			})
		}
	})
}

func actionOther(scheme string) carapace.Action {
	return carapace.ActionMultiPartsN("/", 2, func(c carapace.Context) carapace.Action {
		switch len(c.Parts) {
		case 0:
			hosts, err := man.Hosts(scheme)
			if err != nil {
				return carapace.ActionMessage(err.Error())
			}
			return carapace.ActionValues(hosts...).Suffix("/") // TODO support host-only (no suffix)
		default:
			location, err := man.Location()
			if err != nil {
				return carapace.ActionMessage(err.Error())
			}

			content, err := os.ReadFile(fmt.Sprintf("%v/%v/%v/%v.yaml", location, scheme, c.Parts[0], c.Parts[0])) // TODO move to man
			if err != nil {
				return carapace.ActionMessage(err.Error())
			}

			var entries map[string]string
			if err := yaml.Unmarshal(content, &entries); err != nil {
				return carapace.ActionMessage(err.Error())
			}

			vals := make([]string, 0, len(entries))
			for key := range entries {
				vals = append(vals, key)
			}
			return carapace.ActionValues(vals...)
		}

	})
}
