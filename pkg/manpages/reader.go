package manpages

import (
	"bufio"
	"bytes"
	"io"
)

// source yields logical lines, joining physical lines that end with a
// backslash.
type source struct {
	lines []string
	idx   int
}

func newSource(r io.Reader) (*source, error) {
	scanner := bufio.NewScanner(bufio.NewReader(r))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var joined bytes.Buffer
	res := source{}
	for scanner.Scan() {
		line := scanner.Text()
		if endsWithContinueBackslash(line) {
			joined.WriteString(line[:len(line)-1])
			continue
		}
		if joined.Len() > 0 {
			joined.WriteString(line)
			res.lines = append(res.lines, joined.String())
			joined.Reset()
			continue
		}
		res.lines = append(res.lines, line)
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	if joined.Len() > 0 {
		res.lines = append(res.lines, joined.String())
	}
	return &res, nil
}

func endsWithContinueBackslash(line string) bool {
	count := 0
	for i := len(line) - 1; i >= 0 && line[i] == '\\'; i-- {
		count++
	}
	return count%2 == 1
}

func (s *source) next() (string, bool) {
	if s.idx >= len(s.lines) {
		return "", false
	}
	line := s.lines[s.idx]
	s.idx++
	return line, true
}

func (s *source) peek() (string, bool) {
	if s.idx >= len(s.lines) {
		return "", false
	}
	return s.lines[s.idx], true
}
