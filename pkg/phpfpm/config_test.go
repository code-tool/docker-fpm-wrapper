package phpfpm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The dumps in testdata are the output of "php-fpm -tt" of PHP 5.6, 7.4, and 8.5 for this config:
//
//	[www]
//	listen = /run/php/$pool.sock
//	pm.status_path = /status
//	slowlog = /tmp/$pool.slow
//	request_slowlog_timeout = 3
//	php_value[session.save_path] = "2;/var/lib/php/$pool"
//	php_flag[display_errors] = off
//	php_admin_value[session.gc_maxlifetime] = 3600
//
//	[api]
//	listen = 127.0.0.1:9001
//	php_value[session.gc_maxlifetime] = 60
//	php_admin_value[session.gc_maxlifetime] = 7200
func dumpPools() []Pool {
	return []Pool{
		{
			Name:                     "www",
			Listen:                   "/run/php/www.sock",
			StatusPath:               "/status",
			SlowlogPath:              "/tmp/www.slow",
			RequestSlowlogTimeout:    3 * time.Second,
			RequestSlowlogTraceDepth: 20,
			PhpValues: map[string]string{
				"display_errors":         "0",
				"session.save_path":      "2;/var/lib/php/www",
				"session.gc_maxlifetime": "3600",
			},
		},
		{
			Name:                     "api",
			Listen:                   "127.0.0.1:9001",
			RequestSlowlogTraceDepth: 20,
			PhpValues:                map[string]string{"session.gc_maxlifetime": "7200"},
		},
	}
}

func TestParseDump(t *testing.T) {
	for _, version := range []string{"5.6", "7.4", "8.5"} {
		t.Run("PHP "+version, func(t *testing.T) {
			f, err := os.Open(filepath.Join("testdata", "dump-"+version+".txt"))
			require.NoError(t, err)
			t.Cleanup(func() { _ = f.Close() })

			config, err := ParseDump(f)
			require.NoError(t, err)
			assert.Equal(t, dumpPools(), config.Pools)
		})
	}
}

func TestParseDump_Errors(t *testing.T) {
	const prefix = "[21-Oct-2026 20:14:43] NOTICE: "

	tests := []struct {
		name    string
		dump    string
		wantErr string
	}{
		{
			name:    "empty output",
			dump:    "",
			wantErr: "php-fpm config dump has no pools",
		},
		{
			name:    "global section only",
			dump:    prefix + "[global]\n" + prefix + "\terror_log = /dev/null\n",
			wantErr: "php-fpm config dump has no pools",
		},
		{
			name:    "invalid slowlog timeout",
			dump:    prefix + "[www]\n" + prefix + "\trequest_slowlog_timeout = soon\n",
			wantErr: "pool www: parse request_slowlog_timeout",
		},
		{
			name:    "invalid trace depth",
			dump:    prefix + "[www]\n" + prefix + "\trequest_slowlog_trace_depth = deep\n",
			wantErr: "pool www: parse request_slowlog_trace_depth",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseDump(strings.NewReader(tt.dump))
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestReadConfig(t *testing.T) {
	dump, err := filepath.Abs("testdata/dump-8.5.txt")
	require.NoError(t, err)

	tests := []struct {
		name     string
		script   string
		timeout  time.Duration
		wantArgs string
		wantErr  string
	}{
		{
			name:     "dump of php-fpm",
			script:   "echo \"$*\" > \"$ARGS_FILE\"\ncat '" + dump + "' >&2\n",
			wantArgs: "-d memory_limit=1G --fpm-config /etc/php-fpm.conf -tt",
		},
		{
			name:    "php-fpm rejects the config",
			script:  "echo '[21-Oct-2026 20:14:43] ERROR: failed to open error_log' >&2\nexit 78\n",
			wantErr: "run php-fpm -tt: exit status 78: [21-Oct-2026 20:14:43] ERROR: failed to open error_log",
		},
		{
			name:    "php-fpm does not exit",
			script:  "exec sleep 10\n",
			timeout: 100 * time.Millisecond,
			wantErr: "a FIFO without a reader at error_log or slowlog can block php-fpm",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			argsFile := filepath.Join(dir, "args")
			t.Setenv("ARGS_FILE", argsFile)

			fpmPath := filepath.Join(dir, "php-fpm")
			require.NoError(t, os.WriteFile(fpmPath, []byte("#!/bin/sh\n"+tt.script), 0o755))

			ctx := t.Context()
			if tt.timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.timeout)
				t.Cleanup(cancel)
			}

			config, err := ReadConfig(ctx, fpmPath, "/etc/php-fpm.conf", []string{"-d", "memory_limit=1G"})
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, dumpPools(), config.Pools)

			args, err := os.ReadFile(argsFile)
			require.NoError(t, err)
			assert.Equal(t, tt.wantArgs, strings.TrimSpace(string(args)))
		})
	}
}
