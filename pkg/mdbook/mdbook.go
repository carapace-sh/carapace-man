// Package mdbook converts an archived man page tree into an mdbook: one
// markdown page per man page plus book.toml, an alphabetized SUMMARY.md
// grouped by section, and a report of parse failures and unresolved
// cross references.
package mdbook

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/carapace-sh/carapace-man/pkg/manpages"
)

// Page identifies an archived man page: a plain or gzipped roff source
// following the man(1) naming convention `<name>.<section>[.gz]`.
type Page struct {
	Name    string
	Section string
	Path    string
}

// key is the man:// URI fragment `name(section)`.
func (p Page) key() string {
	return p.Name + "\x00" + p.Section
}

// Load walks an archive directory in man layout (`man[1-8]/*`, plain or
// gzipped) and returns the pages it contains.
func Load(archive string) ([]Page, error) {
	var pages []Page
	err := filepath.WalkDir(archive, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if page, ok := pageFromFile(path); ok {
			pages = append(pages, page)
		}
		return nil
	})
	return pages, err
}

func pageFromFile(path string) (Page, bool) {
	base := filepath.Base(path)
	if strings.HasSuffix(base, ".gz") {
		base = strings.TrimSuffix(base, ".gz")
	}
	dot := strings.LastIndex(base, ".")
	if dot <= 0 || dot == len(base)-1 {
		return Page{}, false
	}
	return Page{Name: base[:dot], Section: base[dot+1:], Path: path}, true
}

// openPage returns a reader over the page source, decompressing gzip pages
// in-process; closer releases both the decompressor and the file.
func openPage(path string) (io.Reader, func(), error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	magic := make([]byte, 2)
	if _, err := io.ReadFull(f, magic); err != nil && err != io.ErrUnexpectedEOF {
		f.Close()
		return nil, nil, err
	}
	if magic[0] == 0x1f && magic[1] == 0x8b {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			f.Close()
			return nil, nil, err
		}
		gz, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			return nil, nil, err
		}
		return gz, func() { gz.Close(); f.Close() }, nil
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}

// manLink matches `(man://topic/section)` URLs emitted by manpages.Parse.
var manLink = regexp.MustCompile(`\(man://([A-Za-z0-9._-]+)/([^)]+)\)`)

// Generate converts an archived man page tree into an mdbook layout:
// `book.toml`, `src/<page>(<section>).md` and an alphabetized SUMMARY.md
// grouped by section. Cross references to pages present in the archive
// become relative markdown links; unresolved ones keep their man:// URL
// and land in the missing-link report, as do parse failures.
func Generate(archive, book string) error {
	pages, err := Load(archive)
	if err != nil {
		return err
	}
	index := make(map[string]Page, len(pages))
	for _, page := range pages {
		if _, known := index[page.key()]; known {
			continue
		}
		index[page.key()] = page
	}

	var problems report
	var entries []Page
	transformed := make(map[string]string, len(pages))
	for _, page := range pages {
		if _, done := transformed[page.key()]; done {
			continue
		}
		source, closer, err := openPage(page.Path)
		if err != nil {
			problems.add("open %v: %v", page.Path, err)
			continue
		}
		out, err := manpages.Parse(source)
		closer()
		if err != nil {
			problems.add("parse %v: %v", page.Path, err)
			continue
		}
		transformed[page.key()] = rewriteLinks(out, page, index, &problems)
		entries = append(entries, page)
	}

	if err := os.MkdirAll(filepath.Join(book, "src"), 0o755); err != nil {
		return err
	}
	for key, markdown := range transformed {
		name, section, _ := strings.Cut(key, "\x00")
		target := filepath.Join(book, "src", fmt.Sprintf("%v(%v).md", name, section))
		if err := os.WriteFile(target, []byte(markdown), 0o644); err != nil {
			return err
		}
	}
	if err := writeBook(book); err != nil {
		return err
	}
	if err := writeSummary(book, entries); err != nil {
		return err
	}
	return writeReport(book, &problems)
}

// rewriteLinks replaces `(man://topic/section)` with relative links to the
// book pages; absent pages are reported and keep their man:// URL.
func rewriteLinks(markdown string, source Page, index map[string]Page, problems *report) string {
	return manLink.ReplaceAllStringFunc(markdown, func(match string) string {
		groups := manLink.FindStringSubmatch(match)
		if groups == nil {
			return match
		}
		target, ok := index[groups[1]+"\x00"+groups[2]]
		if !ok {
			problems.add("missing link %v(%v) in %v(%v)", groups[1], groups[2], source.Name, source.Section)
			return match
		}
		file := fmt.Sprintf("%v(%v).md", target.Name, target.Section)
		if samePage(source, target) {
			problems.add("self link %v(%v) in %v(%v)", groups[1], groups[2], source.Name, source.Section)
		}
		return "(" + fileLinkLabel(file) + ")"
	})
}

func samePage(source, target Page) bool {
	return source.key() == target.key()
}

// report collects parse failures and unresolved cross references.
type report struct {
	rows []string
}

func (r *report) add(format string, args ...interface{}) {
	r.rows = append(r.rows, fmt.Sprintf(format, args...))
}

// writeBook emits the minimal book.toml.
func writeBook(book string) error {
	toml := `[book]
title = "man pages"
src = "src"

[output.html]
git-repository-url = "https://github.com/carapace-sh/carapace-man"
`
	return os.WriteFile(filepath.Join(book, "book.toml"), []byte(toml), 0o644)
}

// sectionRank orders sections: numeric sections first (with suffixed ones
// right after their base), then everything else alphabetically.
func sectionRank(section string) (int, string) {
	if rank, err := strconv.Atoi(section); err == nil {
		return rank, section
	}
	if len(section) > 0 {
		if rank, err := strconv.Atoi(section[:1]); err == nil {
			return rank*100000 + len(section), section
		}
	}
	return -1, section
}

// writeSummary emits the alphabetized SUMMARY.md grouped by section.
func writeSummary(book string, entries []Page) error {
	bySection := map[string][]Page{}
	for _, entry := range entries {
		bySection[entry.Section] = append(bySection[entry.Section], entry)
	}
	type group struct {
		section string
		pages   []Page
		rank    int
	}
	groups := make([]group, 0, len(bySection))
	for section, pages := range bySection {
		rank, _ := sectionRank(section)
		groups = append(groups, group{section: section, pages: pages, rank: rank})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].rank != groups[j].rank {
			return groups[i].rank < groups[j].rank
		}
		return groups[i].section < groups[j].section
	})
	var summary strings.Builder
	summary.WriteString("# Summary\n\n")
	for index, gr := range groups {
		pages := gr.pages
		sort.SliceStable(pages, func(i, j int) bool {
			if pages[i].Name != pages[j].Name {
				return pages[i].Name < pages[j].Name
			}
			return pages[i].Section < pages[j].Section
		})
		if index > 0 {
			summary.WriteString("\n")
		}

		summary.WriteString("## " + gr.section + "\n\n")
		for _, page := range pages {
			summary.WriteString(fmt.Sprintf("- [%v(%v)](<%v(%v).md>)\n", page.Name, page.Section, page.Name, page.Section))
		}
	}
	return os.WriteFile(filepath.Join(book, "src", "SUMMARY.md"), []byte(summary.String()), 0o644)
}

// writeReport emits the parse-failure and missing-link report.
func writeReport(book string, problems *report) error {
	path := filepath.Join(book, "report.txt")
	if len(problems.rows) == 0 {
		if err := os.Remove(path); err != nil && os.IsNotExist(err) == false {
			return err
		}
		return nil
	}
	return os.WriteFile(path, []byte(strings.Join(problems.rows, "\n")+"\n"), 0o644)
}

// fileLinkLabel wraps a destination in angle brackets so parentheses stay
// literal in markdown link targets.
func fileLinkLabel(file string) string {
	return "<" + file + ">"
}
