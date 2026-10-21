package phpfpm

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestErrLogParser_Parse(t *testing.T) {
	ready := ErrLogEntry{
		CreatedAt: time.Date(2026, time.October, 21, 19, 21, 13, 0, time.UTC),
		Level:     LogLevelNotice,
		Message:   "ready to handle connections",
	}
	childSaid := ErrLogEntry{
		CreatedAt: time.Date(2026, time.October, 21, 19, 21, 15, 0, time.UTC),
		Level:     LogLevelWarning,
		Message:   `[pool www] child 17 said into stderr: "worker stderr line"`,
	}

	tests := []struct {
		name         string
		input        string
		wantEntries  []ErrLogEntry
		wantUnparsed string
	}{
		{
			name: "php-fpm lines",
			input: "[21-Oct-2026 19:21:13] NOTICE: ready to handle connections\n" +
				"[21-Oct-2026 19:21:15] WARNING: [pool www] child 17 said into stderr: \"worker stderr line\"\n",
			wantEntries: []ErrLogEntry{ready, childSaid},
		},
		{
			name: "raw worker line between php-fpm lines",
			input: "[21-Oct-2026 19:21:13] NOTICE: ready to handle connections\n" +
				"{\"message\":\"raw worker line\"}\n" +
				"[21-Oct-2026 19:21:15] WARNING: [pool www] child 17 said into stderr: \"worker stderr line\"\n",
			wantEntries:  []ErrLogEntry{ready, childSaid},
			wantUnparsed: "{\"message\":\"raw worker line\"}\n",
		},
		{
			name: "line with invalid timestamp",
			input: "[not a time] NOTICE: ready to handle connections\n" +
				"[21-Oct-2026 19:21:13] NOTICE: ready to handle connections\n",
			wantEntries:  []ErrLogEntry{ready},
			wantUnparsed: "[not a time] NOTICE: ready to handle connections\n",
		},
		{
			name:         "line with unknown level",
			input:        "[21-Oct-2026 19:21:13] TRACE: ready to handle connections\n",
			wantUnparsed: "[21-Oct-2026 19:21:13] TRACE: ready to handle connections\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var unparsed bytes.Buffer
			parser := NewErrLogParser(&unparsed)

			out := make(chan ErrLogEntry, 10)
			err := parser.Parse(t.Context(), strings.NewReader(tt.input), out)
			close(out)

			assert.ErrorIs(t, err, io.EOF)

			var entries []ErrLogEntry
			for entry := range out {
				entries = append(entries, entry)
			}
			assert.Equal(t, tt.wantEntries, entries)
			assert.Equal(t, tt.wantUnparsed, unparsed.String())
		})
	}
}
