//go:build e2e

package e2e

import (
	"encoding/json/v2"
	"strings"
)

// logEntry is a JSON log entry of the wrapper.
type logEntry struct {
	Level    string       `json:"level"`
	Channel  string       `json:"channel"`
	Message  string       `json:"message"`
	Filename string       `json:"filename"`
	Trace    []traceFrame `json:"trace"`
}

// traceFrame is one frame of the trace in a slowlog entry.
type traceFrame struct {
	Path string `json:"path"`
	Func string `json:"func"`
	Line int    `json:"line"`
}

// parseEntries returns the JSON log entries. It ignores the lines that are not JSON objects,
// for example the lines from the log socket and the log pipe.
func parseEntries(lines []string) []logEntry {
	var entries []logEntry
	for _, line := range lines {
		if !strings.HasPrefix(line, "{") {
			continue
		}

		var entry logEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		entries = append(entries, entry)
	}

	return entries
}
