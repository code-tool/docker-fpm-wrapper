//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMetrics examines the metrics that the wrapper reads from the status page of php-fpm.
func TestMetrics(t *testing.T) {
	t.Parallel()

	w := startWrapper(t)

	const requests = 3
	for range requests {
		assert.Equal(t, "ok", w.mustRequest(t, "index.php", ""))
	}

	// The status requests of the wrapper also increase the counter.
	assert.GreaterOrEqual(t, w.poolGauge(t.Context(), t, "phpfpm_accepted_conn", "www"), float64(requests))
	// The config has "pm = static" and "pm.max_children = 2".
	assert.InDelta(t, 2, w.poolGauge(t.Context(), t, "phpfpm_total_processes", "www"), 0)
	assert.InDelta(t, 0, w.poolGauge(t.Context(), t, "phpfpm_slow_requests", "www"), 0)
}
