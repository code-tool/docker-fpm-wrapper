//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAppLogProxy examines the lines that PHP writes to FPM_WRAPPER_SOCK and FPM_WRAPPER_PIPE.
func TestAppLogProxy(t *testing.T) {
	t.Parallel()

	w := startWrapper(t)

	assert.Equal(t, "ok", w.mustRequest(t, "applog.php", ""))

	tests := []struct {
		name string
		line string
	}{
		{name: "socket line", line: "socket line"},
		{name: "socket line with crlf has lf only", line: "socket crlf line"},
		{name: "socket line without newline gets newline", line: "socket line without newline"},
		{name: "socket line after a dropped long line", line: "socket line after long line"},
		{name: "pipe line", line: "pipe line"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w.waitForLine(t, tt.line)
		})
	}

	t.Run("long line is dropped", func(t *testing.T) {
		for _, line := range w.logs.Lines() {
			assert.NotContains(t, line, strings.Repeat("x", 1024))
		}
	})
}
