package main

import (
	"fmt"
	"slices"
	"time"

	"github.com/jessevdk/go-flags"
	_ "github.com/joho/godotenv/autoload"

	"github.com/code-tool/docker-fpm-wrapper/internal/zapx"
)

// Config is the config of the wrapper. A flag takes precedence over its environment variable.
type Config struct {
	LogLevel   zapx.LogLevel `long:"log-level" env:"LOG_LEVEL" default:"-1" description:"Log level: a name (debug, info, warn, error) or a number (-1 is debug)"`
	LogEncoder string        `long:"log-encoder" env:"LOG_ENCODER" default:"auto" choice:"auto" choice:"console" choice:"json" description:"Internal logging encoder"`

	FpmPath       string `short:"f" long:"fpm" env:"FPM" required:"true" description:"Path to php-fpm"`
	FpmConfigPath string `short:"c" long:"fpm-config" env:"FPM_CONFIG" default:"/etc/php/php-fpm.conf" description:"Path to the php-fpm config file"`

	FpmNoErrlogProxy  bool `long:"fpm-no-errlog" env:"FPM_NO_ERRLOG" description:"Disable php-fpm errlog parsing and proxy"`
	FpmNoSlowlogProxy bool `long:"fpm-no-slowlog" env:"FPM_NO_SLOWLOG" description:"Disable php-fpm slowlog parsing and proxy"`

	// Logging proxy section
	WrapperPipe    string `short:"p" long:"wrapper-pipe" env:"WRAPPER_PIPE" default:"/tmp/fpm-wrapper-pipe" description:"Path to the logging pipe, set '' to disable"`
	WrapperSocket  string `short:"s" long:"wrapper-socket" env:"WRAPPER_SOCKET" default:"/tmp/fpm-wrapper.sock" description:"Path to the logging socket, set null to disable"`
	LineBufferSize int    `long:"line-buffer-size" env:"LINE_BUFFER_SIZE" default:"16384" description:"Max log line size (in bytes)"`

	// Prom section
	Listen      string `long:"listen" env:"LISTEN" default:":8080" description:"Address of the Prometheus metrics endpoint"`
	MetricsPath string `long:"metrics-path" env:"METRICS_PATH" default:"/metrics" description:"Path of the Prometheus metrics endpoint"`

	ShutdownDelay time.Duration `long:"shutdown-delay" env:"SHUTDOWN_DELAY" default:"500ms" description:"Delay before process shutdown"`

	SessionCleanupInterval time.Duration `long:"session-cleanup-interval" env:"SESSION_CLEANUP_INTERVAL" default:"0" description:"Interval of the removal of expired session files, 0 to disable"`
}

// createConfig parses the command line and the environment. It returns the arguments after
// "--": the wrapper passes them to php-fpm.
func createConfig(args []string) (*Config, []string, error) {
	var fpmArgs []string
	if i := slices.Index(args, "--"); i >= 0 {
		args, fpmArgs = args[:i], args[i+1:]
	}

	var config Config

	parser := flags.NewParser(&config, flags.HelpFlag)
	rest, err := parser.ParseArgs(args)
	if err != nil {
		return nil, nil, err
	}

	if len(rest) > 0 {
		return nil, nil, fmt.Errorf("unexpected arguments %q: put the arguments of php-fpm after --", rest)
	}

	return &config, fpmArgs, nil
}
