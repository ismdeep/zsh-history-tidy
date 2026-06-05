package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Record is one zsh extended-history entry: ": <timestamp>:<elapsed>;<command>".
// Command may span multiple physical lines when a line ends with a backslash.
type Record struct {
	Timestamp int64
	Elapsed   int
	Command   string
}

// ErrMalformed is returned by parseEntry when a logical entry can't be parsed.
var ErrMalformed = errors.New("malformed history entry")

// Marshal serialises a record back to the zsh extended-history format.
func (r Record) Marshal() string {
	return fmt.Sprintf(": %d:%d;%s", r.Timestamp, r.Elapsed, r.Command)
}

// ParseAll reads a full zsh history stream and returns the parsed records in
// input order. Malformed entries are skipped and counted; the count is
// returned alongside the records so callers can report it.
func ParseAll(r io.Reader) (records []Record, skipped int, err error) {
	scanner := bufio.NewScanner(r)
	// zsh history lines can be long (a pasted heredoc, a huge one-liner).
	// Bump the max token size well above the 64 KiB default.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var entry strings.Builder
	flush := func() {
		if entry.Len() == 0 {
			return
		}
		rec, perr := parseEntry(entry.String())
		entry.Reset()
		if perr != nil {
			skipped++
			return
		}
		records = append(records, rec)
	}

	for scanner.Scan() {
		line := scanner.Text()
		// A new entry starts with ": " and a digit. Anything else is either
		// a continuation of the previous command (line ended with '\') or junk.
		if isEntryStart(line) {
			flush()
			entry.WriteString(line)
			continue
		}
		if entry.Len() == 0 {
			// Stray line with no entry in progress — skip silently.
			skipped++
			continue
		}
		entry.WriteByte('\n')
		entry.WriteString(line)
	}
	flush()

	if err := scanner.Err(); err != nil {
		return nil, skipped, err
	}
	return records, skipped, nil
}

func isEntryStart(line string) bool {
	if !strings.HasPrefix(line, ": ") || len(line) < 4 {
		return false
	}
	// Need at least one digit right after ": ".
	return line[2] >= '0' && line[2] <= '9'
}

// parseEntry parses one logical entry (already joined across continuation lines).
func parseEntry(s string) (Record, error) {
	if !strings.HasPrefix(s, ": ") {
		return Record{}, ErrMalformed
	}
	rest := s[2:]

	colon := strings.IndexByte(rest, ':')
	if colon <= 0 {
		return Record{}, ErrMalformed
	}
	ts, err := strconv.ParseInt(rest[:colon], 10, 64)
	if err != nil {
		return Record{}, ErrMalformed
	}
	rest = rest[colon+1:]

	semi := strings.IndexByte(rest, ';')
	if semi < 0 {
		return Record{}, ErrMalformed
	}
	elapsed, err := strconv.Atoi(rest[:semi])
	if err != nil {
		return Record{}, ErrMalformed
	}

	cmd := rest[semi+1:]
	// Trim only the trailing newline added by continuation joins; keep
	// leading/trailing whitespace inside the command intact.
	cmd = strings.TrimRight(cmd, "\n")

	return Record{Timestamp: ts, Elapsed: elapsed, Command: cmd}, nil
}
