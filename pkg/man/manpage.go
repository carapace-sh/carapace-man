package man

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"strings"

	mantomd "github.com/carapace-sh/carapace-man/third_party/github.com/mle86/man-to-md"
)

func manpage(uid *url.URL) (string, error) {
	args := []string{"--location"}
	if path := strings.TrimPrefix(uid.Path, "/"); path != "" {
		args = append(args, path)
	}
	args = append(args, uid.Host)

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
	for scanner.Scan() { // TODO urks
		line := scanner.Text()
		if found || strings.HasPrefix(line, ".TH") {
			found = true
			filtered.WriteString(line + "\n")
		}
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
