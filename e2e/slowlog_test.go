//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSlowlogProxy examines the slowlog entry that the wrapper writes as JSON.
func TestSlowlogProxy(t *testing.T) {
	t.Parallel()

	w := startWrapper(t)

	assert.Equal(t, "ok", w.mustRequest(t, wwwPort, "slow.php", ""))

	entry := w.waitForEntry(t, func(e logEntry) bool {
		return e.Message == "slowlog"
	})

	assert.Equal(t, "php-fpm", entry.Channel)
	assert.Equal(t, "warn", entry.Level)
	// The wrapper removes the common directory of the script and the trace from all paths.
	assert.Equal(t, "/slow.php", entry.Filename)

	funcs := make([]string, 0, len(entry.Trace))
	for _, frame := range entry.Trace {
		assert.Equal(t, "/slow.php", frame.Path)
		funcs = append(funcs, frame.Func)
	}
	assert.Equal(t, []string{"sleep()", "slowInner()", "slowOuter()"}, funcs)

	assert.InDelta(t, 1, w.poolGauge(t.Context(), t, "phpfpm_slow_requests", "www"), 0)
}
