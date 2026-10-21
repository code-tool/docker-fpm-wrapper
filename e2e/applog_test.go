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

	users := []struct {
		name string
		user string
	}{
		{name: "all processes run as www-data", user: "www-data"},
		// The workers must be able to write to the socket and the pipe that root created.
		{name: "master runs as root, workers run as www-data", user: "root"},
	}
	for _, u := range users {
		t.Run(u.name, func(t *testing.T) {
			t.Parallel()

			w := startWrapperAs(t, u.user)

			assert.Equal(t, "ok", w.mustRequest(t, wwwPort, "applog.php", ""))

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
		})
	}
}
