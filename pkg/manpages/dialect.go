package manpages

import "strings"

// detectMode inspects control-line names to classify the document. The first
// `.TH` selects man(7); the first `.Dd`/`.Dt` selects mdoc(7). In the absence
// of either, man(7) is the fallback.
func detectMode(names []string) mode {
	for _, name := range names {
		switch name {
		case "TH":
			return modeMan
		case "Dd", "Dt":
			return modeMdoc
		}
	}
	return modeMan
}

// sniffDialect scans control-line names from raw line content without
// consuming them.
func sniffDialect(lines []string) mode {
	names := make([]string, 0, len(lines))
	for _, line := range lines {
		stripped, _ := stripComment(line)
		name, _, _, _, ok := parseControl(stripped)
		if !ok {
			trimmed := strings.TrimLeft(stripped, " \t")
			name, _, _, _, ok = parseControl(trimmed)
			if !ok {
				continue
			}
		}
		names = append(names, name)
	}
	return detectMode(names)
}
