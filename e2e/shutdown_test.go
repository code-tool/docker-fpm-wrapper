//go:build e2e

package e2e

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGracefulShutdown sends a stop signal to the wrapper while php-fpm serves a request.
// The wrapper must keep php-fpm running for --shutdown-delay, then stop it gracefully.
func TestGracefulShutdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		signal string
	}{
		{name: "SIGTERM", signal: "TERM"},
		// Docker and Kubernetes send the STOPSIGNAL of the image. The php-fpm images use SIGQUIT.
		{name: "SIGQUIT", signal: "QUIT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			const shutdownDelay = time.Second
			w := startWrapper(t, "--shutdown-delay", shutdownDelay.String())

			var wg sync.WaitGroup

			var slowBody string
			var slowErr error
			wg.Go(func() {
				slowBody, slowErr = w.request("sleep.php", "seconds=3")
			})

			// The status request itself is one active process.
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				assert.GreaterOrEqual(c, w.poolGauge(t.Context(), c, "phpfpm_active_processes", "www"), float64(2))
			}, eventuallyTimeout, eventuallyTick)

			stopStart := time.Now()
			exitCode, _, err := w.ctr.Exec(t.Context(), []string{"sh", "-c", "kill -" + tt.signal + " 1"})
			require.NoError(t, err)
			require.Equal(t, 0, exitCode)

			// php-fpm accepts new requests during the shutdown delay.
			time.Sleep(shutdownDelay / 3)
			assert.Equal(t, "ok", w.mustRequest(t, "index.php", ""))

			wg.Wait()
			require.NoError(t, slowErr)
			assert.Equal(t, "done", slowBody)

			require.EventuallyWithT(t, func(c *assert.CollectT) {
				state, err := w.ctr.State(t.Context())
				require.NoError(c, err)
				require.False(c, state.Running)
				assert.Equal(c, 0, state.ExitCode)
			}, eventuallyTimeout, eventuallyTick)
			assert.GreaterOrEqual(t, time.Since(stopStart), shutdownDelay)
		})
	}
}
