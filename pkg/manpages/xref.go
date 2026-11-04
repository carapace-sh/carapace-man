package manpages

import (
	"regexp"
	"strings"
)

// sectionBody matches man page section tokens carapace's man:// scheme
// resolves: digits 1-9 and 0, n, l, optionally suffixed by p, 3p, 2p.
const sectionBody = `(?:\d|[nl])(?:p|3p|2p)?`

var validSection = regexp.MustCompile("^" + sectionBody + "$")

const xrefTopic = `[A-Za-z0-9_.\-]{1,64}`

// xrefPattern recognizes `topic(section)` with an optionally bolded topic.
var xrefPattern = regexp.MustCompile(
	`(\*{0,2}` + xrefTopic + `\*{0,2})\((` + sectionBody + `)\)`)

// validManSection reports whether a token is a man page section number that
// carapace's man:// scheme resolves.
func validManSection(token string) bool {
	return validSection.MatchString(token)
}

// invalidXref reports whether a reference should stay plain text.
func invalidXref(topic, section string) bool {
	if !validManSection(section) {
		return true
	}
	return strings.ContainsAny(topic, " \t")
}

// linkify converts `topic(section)` occurrences to carapace man:// links,
// keeping the visible text (including existing bold markers) as rendered.
func linkify(text string) string {
	return xrefPattern.ReplaceAllStringFunc(text, func(match string) string {
		groups := xrefPattern.FindStringSubmatch(match)
		if groups == nil {
			return match
		}
		topic := strings.Trim(groups[1], "*")
		if invalidXref(topic, groups[2]) {
			return match
		}
		return "[" + match + "](man://" + topic + "/" + groups[2] + ")"
	})
}
