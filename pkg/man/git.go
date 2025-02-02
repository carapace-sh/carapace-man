package man

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/charmbracelet/lipgloss"
)

type repo struct {
	upstream string
	location string
	progress io.Writer
}

type repoOpt func(c *repo)

func WithLocation(s string) func(c *repo) {
	return func(c *repo) {
		c.location = s
	}

}

func WithProgress(w io.Writer) func(c *repo) {
	return func(c *repo) {
		c.progress = w
	}
}

func NewRepo(opts ...repoOpt) (*repo, error) {
	repo := &repo{
		upstream: "https://github.com/carapace-sh/man",
		progress: io.Discard,
	}
	for _, opt := range opts {
		opt(repo)
	}

	if repo.location == "" {
		var err error
		if repo.location, err = Location(); err != nil {
			return nil, err
		}
	}
	return repo, nil
}

func (r repo) Sync() error {
	if _, err := os.Stat(r.location); err != nil {
		if os.IsNotExist(err) {
			return r.clone()
		}
		return err
	}
	return r.pull()
}

func (r repo) clone() error {
	command := exec.Command("git",
		"clone",
		"--depth", "1",
		"--single-branch",
		"--no-tags",
		r.upstream,
		r.location,
	)
	command.Stdout = r.progress
	command.Stderr = r.progress
	fmt.Fprintln(r.progress, lipgloss.NewStyle().Faint(true).Bold(true).Render(command.String()))
	return command.Run()
}

func (r repo) pull() error {
	command := exec.Command("git",
		"-C", r.location,
		"pull",
	)
	command.Stdout = r.progress
	command.Stderr = r.progress
	fmt.Fprintln(r.progress, lipgloss.NewStyle().Faint(true).Bold(true).Render(command.String()))

	return command.Run()
}
