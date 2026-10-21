//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// phpVersionDecorateWorkersOutput is the first PHP version with decorate_workers_output.
const phpVersionDecorateWorkersOutput = 70300

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

// readPHPVersionID runs PHP in the image and returns its PHP_VERSION_ID.
func readPHPVersionID(ctx context.Context, image string) (int, error) {
	ctr, err := testcontainers.Run(ctx, image,
		testcontainers.WithCmd("php", "-r", "echo PHP_VERSION_ID;"),
		testcontainers.WithWaitStrategy(wait.ForExit()),
	)
	if ctr != nil {
		defer func() { _ = ctr.Terminate(context.WithoutCancel(ctx)) }()
	}
	if err != nil {
		return 0, fmt.Errorf("run php: %w", err)
	}

	logs, err := ctr.Logs(ctx)
	if err != nil {
		return 0, fmt.Errorf("read the output of php: %w", err)
	}
	defer func() { _ = logs.Close() }()

	out, err := io.ReadAll(logs)
	if err != nil {
		return 0, fmt.Errorf("read the output of php: %w", err)
	}

	versionID, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("parse PHP_VERSION_ID: %w", err)
	}

	return versionID, nil
}

// renderFpmConfig renders testdata/php-fpm.conf.tmpl for the PHP version into output.
func renderFpmConfig(output string, phpVersionID int) error {
	tmpl, err := template.ParseFiles("testdata/php-fpm.conf.tmpl")
	if err != nil {
		return fmt.Errorf("parse the php-fpm config template: %w", err)
	}

	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create the php-fpm config: %w", err)
	}
	defer func() { _ = f.Close() }()

	data := struct{ DecorateWorkersOutput bool }{
		DecorateWorkersOutput: phpVersionID >= phpVersionDecorateWorkersOutput,
	}
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("render the php-fpm config: %w", err)
	}

	return f.Close()
}

// suite holds the facts that TestMain finds once for all tests.
type suite struct {
	wrapperBinary string // the path of the wrapper binary on the host
	fpmConfig     string // the path of the rendered php-fpm config on the host
	phpVersionID  int    // PHP_VERSION_ID of the image
}

// newSuite builds the wrapper and prepares the php-fpm config for image in dir.
func newSuite(ctx context.Context, dir, image string) (suite, error) {
	s := suite{
		wrapperBinary: filepath.Join(dir, "docker-fpm-wrapper"),
		fpmConfig:     filepath.Join(dir, "php-fpm.conf"),
	}

	if err := buildWrapper(s.wrapperBinary); err != nil {
		return suite{}, err
	}

	versionID, err := readPHPVersionID(ctx, image)
	if err != nil {
		return suite{}, err
	}
	s.phpVersionID = versionID

	if err := renderFpmConfig(s.fpmConfig, versionID); err != nil {
		return suite{}, err
	}

	return s, nil
}

// hasDecorateWorkersOutput tells if php-fpm of the image has decorate_workers_output.
func (s suite) hasDecorateWorkersOutput() bool {
	return s.phpVersionID >= phpVersionDecorateWorkersOutput
}
