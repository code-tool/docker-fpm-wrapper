//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestOfficialImageConfig runs the wrapper with the default php-fpm config of the image. The
// config has error_log = /proc/self/fd/2 in an included file, and PHP 7.4 and earlier use a
// relative include.
func TestOfficialImageConfig(t *testing.T) {
	t.Parallel()

	logs := newLogCollector()
	ctr := runContainer(t, "www-data", wait.ForListeningPort(wwwPort), logs,
		"--fpm", fpmPath, "--fpm-config", "/usr/local/etc/php-fpm.conf")
	w := &wrapper{ctr: ctr, logs: logs}

	// The wrapper proxies the error log: the entries of php-fpm are JSON.
	w.waitForEntry(t, func(e logEntry) bool {
		return e.Channel == "php-fpm" && e.Level == "info" && strings.Contains(e.Message, "ready to handle connections")
	})
}

// TestRestart restarts the container. The FIFOs and the configs of the first start stay in
// the run dir of the wrapper.
func TestRestart(t *testing.T) {
	t.Parallel()

	w := startWrapper(t)
	assert.Equal(t, "ok", w.mustRequest(t, wwwPort, "index.php", ""))

	require.NoError(t, w.ctr.Stop(t.Context(), new(15*time.Second)))
	require.NoError(t, w.ctr.Start(t.Context()))
	w.readEndpoints(t)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		body, err := w.request(wwwPort, "index.php", "")
		require.NoError(c, err)
		assert.Equal(c, "ok", body)
	}, eventuallyTimeout, eventuallyTick)

	// php-fpm reported "ready to handle connections" through the error log proxy after each start.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		ready := 0
		for _, entry := range parseEntries(readLogs(t.Context(), c, w.ctr)) {
			if entry.Channel == "php-fpm" && strings.Contains(entry.Message, "ready to handle connections") {
				ready++
			}
		}
		assert.Equal(c, 2, ready)
	}, eventuallyTimeout, eventuallyTick)
}
