package man

import "strings"

// WINDOWS file systems reject '<', '>', ':', '"', '/', '\', '|', '?', '*' in
// file names (see https://learn.microsoft.com/de-de/windows/win32/fileio/naming-a-file).
// Command paths may legitimately contain such characters (e.g. mercurial's
// `hg debug::clonebundle-manifest`), so names used as file names get
// percent-encoded (`:` → `%3A`) by MangleFilename.
// UnmangleFilename reverses it. Both functions are mutually inverse.

var percent = [][2]string{
	{"%", "%25"},
	{"<", "%3C"},
	{">", "%3E"},
	{":", "%3A"},
	{"\"", "%22"},
	{"\\", "%5C"},
	{"|", "%7C"},
	{"?", "%3F"},
	{"*", "%2A"},
}

func MangleFilename(name string) string {
	for _, p := range percent {
		name = strings.ReplaceAll(name, p[0], p[1])
	}
	return name
}

func UnmangleFilename(name string) string {
	for _, p := range percent {
		name = strings.ReplaceAll(name, p[1], p[0])
	}
	return name
}
