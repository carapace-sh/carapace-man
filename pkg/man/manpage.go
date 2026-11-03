package man

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/carapace-sh/carapace-man/pkg/manpages"
	mantomd "github.com/carapace-sh/carapace-man/third_party/github.com/mle86/man-to-md"
)

// ManToMd renders a system man page as markdown through the native
// manpages.Parse converter. The `man` binary is only used to locate the page.
func ManToMd(name string, section string) (string, error) {
	path, err := manLocation(name, section)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var reader io.Reader = f
	magic := make([]byte, 2)
	if n, _ := io.ReadFull(f, magic); n == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
		gz, err := gzip.NewReader(f)
		if err != nil {
			return "", err
		}
		defer gz.Close()
		reader = gz
	}

	return manpages.Parse(reader)
}

// ManToMdLegacy renders a system man page through the vendored GPL Perl
// script; kept for one release cycle as the --legacy escape hatch.
func ManToMdLegacy(name string, section string) (string, error) {
	args := []string{"--location"}
	if section != "" {
		args = append(args, section)
	}
	args = append(args, name)

	stderr := &bytes.Buffer{}
	command := exec.Command("man", args...)
	command.Stderr = stderr
	output, err := command.Output()
	if err != nil {
		return "", errors.New(stderr.String())
	}

	f, err := os.Open(strings.SplitN(string(output), "\n", 2)[0])
	if err != nil {
		return "", err
	}
	defer f.Close()

	r, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer r.Close()

	tempfile, err := os.CreateTemp(os.TempDir(), "man-to-md")
	if err != nil {
		return "", err
	}
	defer tempfile.Close()
	defer os.Remove(tempfile.Name())

	if err = os.WriteFile(tempfile.Name(), []byte(mantomd.Script), os.ModePerm); err != nil {
		return "", err
	}

	filtered := &bytes.Buffer{}
	found := false
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if found || strings.HasPrefix(line, ".TH") {
			found = true
			filtered.WriteString(line)
			filtered.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	stderr = &bytes.Buffer{}
	command = exec.Command("perl", tempfile.Name())
	command.Stdin = filtered
	command.Stderr = stderr
	output, err = command.Output()
	if err != nil {
		return "", errors.New(stderr.String())
	}

	return string(output), nil
}

func manLocation(name, section string) (string, error) {
	args := []string{"--location"}
	if section != "" {
		args = append(args, section)
	}
	args = append(args, name)

	stderr := &bytes.Buffer{}
	command := exec.Command("man", args...)
	command.Stderr = stderr
	output, err := command.Output()
	if err != nil {
		return "", errors.New(stderr.String())
	}
	return strings.SplitN(string(output), "\n", 2)[0], nil
}

func manpage(uid *url.URL) (string, error) {
	section := ""
	if path := strings.TrimPrefix(uid.Path, "/"); path != "" {
		section = path
	}
	return ManToMd(uid.Host, section)
}
