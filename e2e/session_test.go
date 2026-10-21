//go:build e2e

package e2e

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSessionCleanup examines the removal of expired session files. The "www" pool has
// session.gc_maxlifetime = 2 and session.save_path = /tmp/sessions.
func TestSessionCleanup(t *testing.T) {
	t.Parallel()

	const (
		closedSession = "/tmp/sessions/sess_closed"
		openSession   = "/tmp/sessions/sess_open"
	)

	w := startWrapper(t, "--session-cleanup-interval", "500ms")

	assert.Equal(t, "ok", w.mustRequest(t, wwwPort, "session.php", "id=closed"))

	// This request keeps its session file open for 6 s, longer than the lifetime.
	var wg sync.WaitGroup
	var openBody string
	var openErr error
	wg.Go(func() {
		openBody, openErr = w.request(wwwPort, "session.php", "id=open&seconds=6")
	})

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.True(c, w.fileExists(t.Context(), c, openSession))
	}, eventuallyTimeout, eventuallyTick)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.False(c, w.fileExists(t.Context(), c, closedSession), "the expired session file must be removed")
	}, eventuallyTimeout, eventuallyTick)

	// Three more cleanups run, and the open session file is older than the lifetime.
	time.Sleep(1500 * time.Millisecond)
	assert.True(t, w.fileExists(t.Context(), t, openSession), "the open session file must stay")

	wg.Wait()
	require.NoError(t, openErr)
	assert.Equal(t, "ok", openBody)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.False(c, w.fileExists(t.Context(), c, openSession), "the session file must be removed after it expires")
	}, eventuallyTimeout, eventuallyTick)
}
