//go:build e2e

package e2e

import (
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// readLogs reads all output of a container that exited.
func readLogs(t *testing.T, ctr *testcontainers.DockerContainer) []string {
	t.Helper()

	reader, err := ctr.Logs(t.Context())
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	require.NoError(t, err)

	return strings.Split(string(data), "\n")
}

// TestStartFailure examines the exit code and the output of the wrapper when it cannot start php-fpm.
func TestStartFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		wantExitCode int
		wantOutput   string // a part of a line of the output
	}{
		{
			name:         "fpm path is not set",
			args:         []string{"--fpm-config", fpmConfigPath},
			wantExitCode: 1,
			wantOutput:   "the required flag `-f, --fpm' was not specified",
		},
		{
			name:         "fpm config does not exist",
			args:         []string{"--fpm", fpmPath, "--fpm-config", "/nonexistent/php-fpm.conf"},
			wantExitCode: 1,
			wantOutput:   `"message":"Can't parse fpm config"`,
		},
		{
			name:         "fpm binary does not exist",
			args:         []string{"--fpm", "/nonexistent/php-fpm", "--fpm-config", fpmConfigPath},
			wantExitCode: 1,
			wantOutput:   `"message":"Can't start php-fpm"`,
		},
		{
			// The wrapper exits with the exit code of php-fpm: 78 is EX_CONFIG.
			name:         "php-fpm rejects the config",
			args:         []string{"--fpm", fpmPath, "--fpm-config", invalidFpmConfigPath},
			wantExitCode: 78,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctr := runContainer(t, "www-data", wait.ForExit(), newLogCollector(), tt.args...)

			state, err := ctr.State(t.Context())
			require.NoError(t, err)
			assert.Equal(t, tt.wantExitCode, state.ExitCode)

			if tt.wantOutput == "" {
				return
			}

			lines := readLogs(t, ctr)
			assert.True(t, slices.ContainsFunc(lines, func(line string) bool {
				return strings.Contains(line, tt.wantOutput)
			}), "no line with %q in the output:\n%s", tt.wantOutput, strings.Join(lines, "\n"))
		})
	}
}
