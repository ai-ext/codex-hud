package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// maxLineSize is the maximum line length that the scanner will accept (1 MB).
const maxLineSize = 1 << 20 // 1 048 576 bytes

// ParseLine parses a single JSONL line into an Event. Leading and trailing
// whitespace is trimmed before unmarshalling.
func ParseLine(line string) (*Event, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil, fmt.Errorf("empty line")
	}

	var ev Event
	if err := json.Unmarshal([]byte(trimmed), &ev); err != nil {
		return nil, fmt.Errorf("parsing JSONL line: %w", err)
	}
	return &ev, nil
}

// ParseLines reads all lines from r, parsing each non-blank line as a JSONL
// Event. Lines that fail to parse are collected into the errors slice; they do
// not stop processing.
func ParseLines(r io.Reader) ([]*Event, []error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, maxLineSize), maxLineSize)

	var events []*Event
	var errs []error

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		ev, err := ParseLine(line)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		events = append(events, ev)
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Errorf("scanner error: %w", err))
	}

	return events, errs
}
