//go:build e2e

// Package e2e runs docker-fpm-wrapper with php-fpm in a container and examines what the wrapper
// writes to its output.
package e2e

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

const (
	// defaultPHPImage is the image for the tests. Set E2E_PHP_IMAGE to use a different image.
	defaultPHPImage = "php:8.5-fpm-alpine"

	wrapperPath          = "/usr/local/bin/docker-fpm-wrapper"
	fpmPath              = "/usr/local/sbin/php-fpm"
	fpmConfigPath        = "/usr/local/etc/php-fpm-e2e.conf"
	invalidFpmConfigPath = "/usr/local/etc/php-fpm-e2e-invalid.conf"
	scriptDir            = "/var/www/html"

	// wwwPort and rawPort are the FastCGI ports of the pools "www" and "raw".
	wwwPort     = "9000/tcp"
	rawPort     = "9001/tcp"
	metricsPort = "8080/tcp"

	eventuallyTimeout = 10 * time.Second
	eventuallyTick    = 50 * time.Millisecond
)

// wrapperBinary is the path of the wrapper binary on the host. TestMain builds the binary once
// for all tests, and the go tool gives no other way to share it with the tests.
var wrapperBinary string //nolint:gochecknoglobals // TestMain sets it before the tests start.

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	dir, err := os.MkdirTemp("", "fpm-wrapper-e2e")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create a temp dir: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()

	wrapperBinary = filepath.Join(dir, "docker-fpm-wrapper")
	if err := buildWrapper(wrapperBinary); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return m.Run()
}

// buildWrapper builds a static Linux binary for the architecture of the host.
func buildWrapper(output string) error {
	cmd := exec.Command("go", "build", "-o", output, "../cmd/docker-fpm-wrapper")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH="+runtime.GOARCH)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build the wrapper: %w", err)
	}

	return nil
}

func phpImage() string {
	return cmp.Or(os.Getenv("E2E_PHP_IMAGE"), defaultPHPImage)
}
