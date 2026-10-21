//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	fcgiclient "github.com/tomasen/fcgi_client"
)

// containerFiles returns the wrapper binary, the php-fpm configs, and the PHP scripts.
func containerFiles(t *testing.T) []testcontainers.ContainerFile {
	t.Helper()

	files := []testcontainers.ContainerFile{
		{HostFilePath: wrapperBinary, ContainerFilePath: wrapperPath, FileMode: 0o755},
		{HostFilePath: "testdata/php-fpm.conf", ContainerFilePath: fpmConfigPath, FileMode: 0o644},
		{HostFilePath: "testdata/invalid-php-fpm.conf", ContainerFilePath: invalidFpmConfigPath, FileMode: 0o644},
	}

	scripts, err := os.ReadDir("testdata/www")
	require.NoError(t, err)

	for _, script := range scripts {
		files = append(files, testcontainers.ContainerFile{
			HostFilePath:      filepath.Join("testdata/www", script.Name()),
			ContainerFilePath: path.Join(scriptDir, script.Name()),
			FileMode:          0o644,
		})
	}

	return files
}

// runContainer starts the wrapper as the entrypoint of a php-fpm container that runs as user.
func runContainer(
	t *testing.T, user string, strategy wait.Strategy, logs testcontainers.LogConsumer, args ...string,
) *testcontainers.DockerContainer {
	t.Helper()

	ctr, err := testcontainers.Run(t.Context(), phpImage(),
		testcontainers.WithFiles(containerFiles(t)...),
		testcontainers.WithEntrypoint(wrapperPath),
		testcontainers.WithCmd(args...),
		testcontainers.WithExposedPorts(wwwPort, rawPort, metricsPort),
		testcontainers.WithConfigModifier(func(config *container.Config) {
			config.User = user
		}),
		testcontainers.WithLogConsumers(logs),
		testcontainers.WithWaitStrategy(strategy),
	)
	testcontainers.CleanupContainer(t, ctr)
	require.NoError(t, err)

	return ctr
}

// wrapper is a running container with the wrapper and php-fpm.
type wrapper struct {
	ctr  *testcontainers.DockerContainer
	logs *logCollector

	fcgiAddrs  map[string]string // the host address of each FastCGI port
	metricsURL string
}

// startWrapper starts the wrapper as www-data, as a container in Kubernetes usually runs.
func startWrapper(t *testing.T, extraArgs ...string) *wrapper {
	t.Helper()

	return startWrapperAs(t, "www-data", extraArgs...)
}

// startWrapperAs starts the wrapper as user with the e2e php-fpm config and waits until php-fpm
// and the metrics endpoint are ready. The extra arguments go after the default arguments.
func startWrapperAs(t *testing.T, user string, extraArgs ...string) *wrapper {
	t.Helper()

	args := append([]string{"--fpm", fpmPath, "--fpm-config", fpmConfigPath, "--listen", ":8080"}, extraArgs...)
	strategy := wait.ForAll(
		wait.ForListeningPort(wwwPort),
		wait.ForListeningPort(rawPort),
		wait.ForHTTP("/metrics").WithPort(metricsPort),
	)

	logs := newLogCollector()
	ctr := runContainer(t, user, strategy, logs, args...)

	fcgiAddrs := make(map[string]string)
	for _, port := range []string{wwwPort, rawPort} {
		addr, err := ctr.PortEndpoint(t.Context(), port, "")
		require.NoError(t, err)

		fcgiAddrs[port] = addr
	}

	metricsEndpoint, err := ctr.PortEndpoint(t.Context(), metricsPort, "http")
	require.NoError(t, err)

	return &wrapper{ctr: ctr, logs: logs, fcgiAddrs: fcgiAddrs, metricsURL: metricsEndpoint + "/metrics"}
}

// request sends a FastCGI GET request for a script in scriptDir to the pool on port and returns
// the body of the response.
func (w *wrapper) request(port, script, query string) (string, error) {
	client, err := fcgiclient.DialTimeout("tcp", w.fcgiAddrs[port], time.Second)
	if err != nil {
		return "", fmt.Errorf("dial php-fpm: %w", err)
	}
	defer client.Close()

	resp, err := client.Get(map[string]string{
		"QUERY_STRING":    query,
		"SCRIPT_FILENAME": path.Join(scriptDir, script),
		"SCRIPT_NAME":     "/" + script,
	})
	if err != nil {
		return "", fmt.Errorf("send the request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read the response: %w", err)
	}

	return string(body), nil
}

func (w *wrapper) mustRequest(t *testing.T, port, script, query string) string {
	t.Helper()

	body, err := w.request(port, script, query)
	require.NoError(t, err)

	return body
}

// fileExists tells if a regular file exists in the container.
// It takes require.TestingT, so that a condition of require.EventuallyWithT can call it.
func (w *wrapper) fileExists(ctx context.Context, t require.TestingT, path string) bool {
	if h, ok := t.(interface{ Helper() }); ok {
		h.Helper()
	}

	exitCode, _, err := w.ctr.Exec(ctx, []string{"test", "-f", path})
	require.NoError(t, err)

	return exitCode == 0
}

// waitForLine waits until the output of the wrapper has a line that is equal to line.
func (w *wrapper) waitForLine(t *testing.T, line string) {
	t.Helper()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Contains(c, w.logs.Lines(), line)
	}, eventuallyTimeout, eventuallyTick)
}

// waitForEntry waits until the output of the wrapper has a JSON log entry that agrees with match.
func (w *wrapper) waitForEntry(t *testing.T, match func(logEntry) bool) logEntry {
	t.Helper()

	var found logEntry
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		entries := parseEntries(w.logs.Lines())
		i := slices.IndexFunc(entries, match)
		if assert.GreaterOrEqual(c, i, 0, "no matching log entry") {
			found = entries[i]
		}
	}, eventuallyTimeout, eventuallyTick)

	return found
}

// poolGauge scrapes the metrics endpoint and returns the value of a gauge for the pool.
// It takes require.TestingT, so that a condition of require.EventuallyWithT can call it.
func (w *wrapper) poolGauge(ctx context.Context, t require.TestingT, name, pool string) float64 {
	if h, ok := t.(interface{ Helper() }); ok {
		h.Helper()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.metricsURL, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	parser := expfmt.NewTextParser(model.UTF8Validation)
	families, err := parser.TextToMetricFamilies(resp.Body)
	require.NoError(t, err)

	family, ok := families[name]
	require.True(t, ok, "metric %s is missing", name)

	for _, metric := range family.GetMetric() {
		for _, label := range metric.GetLabel() {
			if label.GetName() == "pool_name" && label.GetValue() == pool {
				return metric.GetGauge().GetValue()
			}
		}
	}

	require.FailNow(t, "no value for the pool", "metric %s, pool %s", name, pool)

	return 0
}
