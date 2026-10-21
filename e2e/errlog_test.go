//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestErrorLogProxy examines the entries of the php-fpm error log that the wrapper writes as JSON.
func TestErrorLogProxy(t *testing.T) {
	t.Parallel()

	w := startWrapper(t)

	// The "raw" pool writes a line in an unknown format to the error log before the "www" pool
	// writes its line. The wrapper must write the raw line without a change and keep parsing.
	assert.Equal(t, "ok", w.mustRequest(t, rawPort, "stderr.php", "message=raw+stderr+line"))
	w.waitForLine(t, "raw stderr line")

	assert.Equal(t, "ok", w.mustRequest(t, wwwPort, "stderr.php", ""))

	tests := []struct {
		name      string
		level     string
		inMessage string
	}{
		{name: "notice of php-fpm is info", level: "info", inMessage: "ready to handle connections"},
		{name: "stderr of a worker is warn", level: "warn", inMessage: "worker stderr line"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w.waitForEntry(t, func(e logEntry) bool {
				return e.Channel == "php-fpm" && e.Level == tt.level && strings.Contains(e.Message, tt.inMessage)
			})
		})
	}
}
