package main

import (
	"errors"
	"testing"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/code-tool/docker-fpm-wrapper/internal/zapx"
)

// TestCreateConfig sets environment variables, so it does not run in parallel.
func TestCreateConfig(t *testing.T) {
	defaults := Config{
		LogLevel:       zapx.LogLevel(zapcore.DebugLevel),
		LogEncoder:     "auto",
		FpmPath:        "/usr/sbin/php-fpm",
		FpmConfigPath:  "/etc/php/php-fpm.conf",
		RunDir:         "/tmp/docker-fpm-wrapper",
		WrapperPipe:    "/tmp/fpm-wrapper-pipe",
		WrapperSocket:  "/tmp/fpm-wrapper.sock",
		LineBufferSize: 16 * 1024,
		Listen:         ":8080",
		MetricsPath:    "/metrics",
		ShutdownDelay:  500 * time.Millisecond,
	}
	with := func(change func(c *Config)) Config {
		c := defaults
		change(&c)
		return c
	}

	tests := []struct {
		name        string
		args        []string
		env         map[string]string
		want        Config
		wantFpmArgs []string
		wantErr     string
	}{
		{
			name: "defaults",
			args: []string{"-f", "/usr/sbin/php-fpm"},
			want: defaults,
		},
		{
			name: "flags",
			args: []string{
				"--fpm", "/usr/sbin/php-fpm", "-c", "/etc/fpm.conf", "--run-dir", "/run/wrapper",
				"--log-level", "-1", "--log-encoder", "json",
				"--fpm-no-errlog", "--fpm-no-slowlog",
				"-p", "", "-s", "null", "--line-buffer-size", "1024",
				"--listen", ":9090", "--metrics-path", "/m",
				"--shutdown-delay", "2s", "--session-cleanup-interval", "30m",
			},
			want: Config{
				LogLevel:               zapx.LogLevel(zapcore.DebugLevel),
				LogEncoder:             "json",
				FpmPath:                "/usr/sbin/php-fpm",
				FpmConfigPath:          "/etc/fpm.conf",
				RunDir:                 "/run/wrapper",
				FpmNoErrlogProxy:       true,
				FpmNoSlowlogProxy:      true,
				WrapperPipe:            "",
				WrapperSocket:          "null",
				LineBufferSize:         1024,
				Listen:                 ":9090",
				MetricsPath:            "/m",
				ShutdownDelay:          2 * time.Second,
				SessionCleanupInterval: 30 * time.Minute,
			},
		},
		{
			name: "environment",
			env: map[string]string{
				"FPM":                      "/usr/sbin/php-fpm",
				"LOG_LEVEL":                "info",
				"FPM_NO_ERRLOG":            "true",
				"WRAPPER_PIPE":             "",
				"SHUTDOWN_DELAY":           "1s",
				"SESSION_CLEANUP_INTERVAL": "30m",
			},
			want: with(func(c *Config) {
				c.LogLevel = zapx.LogLevel(zapcore.InfoLevel)
				c.FpmNoErrlogProxy = true
				c.WrapperPipe = ""
				c.ShutdownDelay = time.Second
				c.SessionCleanupInterval = 30 * time.Minute
			}),
		},
		{
			name: "flag takes precedence over environment",
			args: []string{"-f", "/usr/sbin/php-fpm", "--log-level", "error"},
			env:  map[string]string{"LOG_LEVEL": "info"},
			want: with(func(c *Config) { c.LogLevel = zapx.LogLevel(zapcore.ErrorLevel) }),
		},
		{
			name:        "arguments after double dash go to php-fpm",
			args:        []string{"-f", "/usr/sbin/php-fpm", "--", "-d", "memory_limit=1G", "--log-level", "info"},
			want:        defaults,
			wantFpmArgs: []string{"-d", "memory_limit=1G", "--log-level", "info"},
		},
		{
			name:    "php-fpm flag before double dash",
			args:    []string{"-f", "/usr/sbin/php-fpm", "--nodaemonize=false", "memory_limit=1G", "--", "-y", "/etc/fpm.conf"},
			wantErr: "unknown flag `nodaemonize'",
		},
		{
			name:    "positional argument before double dash",
			args:    []string{"-f", "/usr/sbin/php-fpm", "memory_limit=1G", "--", "-y", "/etc/fpm.conf"},
			wantErr: `unexpected arguments ["memory_limit=1G"]: put the arguments of php-fpm after --`,
		},
		{
			name:    "fpm path is not set",
			wantErr: "the required flag `-f, --fpm' was not specified",
		},
		{
			name:    "unknown log encoder",
			args:    []string{"-f", "/usr/sbin/php-fpm", "--log-encoder", "xml"},
			wantErr: "Invalid value `xml' for option `--log-encoder'",
		},
		{
			name:    "invalid log level",
			args:    []string{"-f", "/usr/sbin/php-fpm", "--log-level", "verbose"},
			wantErr: `unknown log level "verbose"`,
		},
		{
			name:    "invalid duration in environment",
			args:    []string{"-f", "/usr/sbin/php-fpm"},
			env:     map[string]string{"SHUTDOWN_DELAY": "soon"},
			wantErr: `time: invalid duration "soon"`,
		},
		{
			name:    "unknown flag",
			args:    []string{"-f", "/usr/sbin/php-fpm", "--fpm-pool", "www"},
			wantErr: "unknown flag `fpm-pool'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for name, value := range tt.env {
				t.Setenv(name, value)
			}

			got, fpmArgs, err := createConfig(tt.args)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, *got)
			if len(tt.wantFpmArgs) == 0 {
				assert.Empty(t, fpmArgs)
			} else {
				assert.Equal(t, tt.wantFpmArgs, fpmArgs)
			}
		})
	}

	t.Run("help", func(t *testing.T) {
		_, _, err := createConfig([]string{"--help"})

		flagsErr, ok := errors.AsType[*flags.Error](err)
		require.True(t, ok)
		assert.Equal(t, flags.ErrHelp, flagsErr.Type)
		assert.Contains(t, flagsErr.Message, "--session-cleanup-interval=")
		assert.Contains(t, flagsErr.Message, "[$SESSION_CLEANUP_INTERVAL]")
	})
}
