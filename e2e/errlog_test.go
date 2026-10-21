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

	assert.Equal(t, "ok", w.mustRequest(t, "stderr.php", ""))

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
