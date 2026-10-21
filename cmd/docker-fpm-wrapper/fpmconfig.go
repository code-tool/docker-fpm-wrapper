package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

const (
	// fpmConfigReadTimeout limits "php-fpm -tt". It blocks on a FIFO without a reader.
	fpmConfigReadTimeout = 30 * time.Second

	fpmTestConfigName = "php-fpm-test.conf"
	fpmConfigName     = "php-fpm.conf"
	fpmErrorLogName   = "error.log"
)

// fpmSetup tells how the wrapper runs php-fpm.
type fpmSetup struct {
	// configPath is the overlay config for "php-fpm --fpm-config".
	configPath string
	// errorLogPath is the FIFO for error_log. It is empty when the wrapper does not proxy the error log.
	errorLogPath string
	// pools are the pools of php-fpm. SlowlogPath of a pool is the FIFO of the wrapper. It is
	// empty when the wrapper does not proxy the slowlog of the pool.
	pools []phpfpm.Pool
}

// prepareFpmConfig reads the php-fpm config of the user and writes an overlay config to runDir.
// The overlay moves error_log and the slowlog of each pool to FIFOs in runDir. Then the wrapper
// does not change the files of the user, and php-fpm does not need write access to their
// directories.
func prepareFpmConfig(
	ctx context.Context,
	fpmPath, userConfigPath string, fpmArgs []string,
	runDir string,
	proxyErrorLog, proxySlowlog bool,
) (fpmSetup, error) {
	userConfigPath, err := filepath.Abs(userConfigPath)
	if err != nil {
		return fpmSetup{}, fmt.Errorf("find php-fpm config: %w", err)
	}

	// php-fpm ignores an include without matching files, so check the file here.
	if _, err := os.Stat(userConfigPath); err != nil {
		return fpmSetup{}, fmt.Errorf("find php-fpm config: %w", err)
	}

	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return fpmSetup{}, fmt.Errorf("create run dir: %w", err)
	}

	// "php-fpm -tt" opens error_log. /dev/null keeps it away from the files and FIFOs of the
	// user. php-fpm writes the dump at the notice level.
	testOverlay := phpfpm.Overlay{
		Include: userConfigPath,
		Global:  map[string]string{"error_log": "/dev/null", "log_level": "notice"},
	}
	testConfigPath := filepath.Join(runDir, fpmTestConfigName)
	if err := testOverlay.WriteFile(testConfigPath); err != nil {
		return fpmSetup{}, err
	}

	readCtx, cancel := context.WithTimeout(ctx, fpmConfigReadTimeout)
	defer cancel()

	fpmConfig, err := phpfpm.ReadConfig(readCtx, fpmPath, testConfigPath, fpmArgs)
	if err != nil {
		return fpmSetup{}, err
	}

	setup := fpmSetup{configPath: filepath.Join(runDir, fpmConfigName), pools: fpmConfig.Pools}
	overlay := phpfpm.Overlay{
		Include: userConfigPath,
		Global:  make(map[string]string),
		Pools:   make(map[string]map[string]string),
	}

	if proxyErrorLog {
		setup.errorLogPath = filepath.Join(runDir, fpmErrorLogName)
		overlay.Global["error_log"] = setup.errorLogPath
	}

	for i := range setup.pools {
		pool := &setup.pools[i]

		// php-fpm writes the slowlog only when request_slowlog_timeout is set.
		if !proxySlowlog || pool.RequestSlowlogTimeout == 0 {
			pool.SlowlogPath = ""
			continue
		}

		pool.SlowlogPath = filepath.Join(runDir, "slowlog-"+url.PathEscape(pool.Name)+".log")
		overlay.Pools[pool.Name] = map[string]string{"slowlog": pool.SlowlogPath}
	}

	if err := overlay.WriteFile(setup.configPath); err != nil {
		return fpmSetup{}, err
	}

	return setup, nil
}
