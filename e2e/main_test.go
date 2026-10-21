//go:build e2e

// Package e2e runs docker-fpm-wrapper with php-fpm in a container and examines what the wrapper
// writes to its output.
package e2e

import (
	"cmp"
	"context"
	"fmt"
	"os"
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

// env holds the facts that TestMain finds once for all tests. The go tool gives no other way
// to share them with the tests.
var env suite //nolint:gochecknoglobals // TestMain sets it before the tests start.

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

	env, err = newSuite(context.Background(), dir, phpImage())
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return m.Run()
}

func phpImage() string {
	return cmp.Or(os.Getenv("E2E_PHP_IMAGE"), defaultPHPImage)
}
