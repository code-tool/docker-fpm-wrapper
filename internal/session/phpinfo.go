package session

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strings"
)

// parsePhpinfo returns the local values of the directives in the text output of phpinfo().
// A directive line has the form "name => local value => master value".
func parsePhpinfo(r io.Reader) (map[string]string, error) {
	directives := make(map[string]string)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), " => ")
		if len(parts) != 3 {
			continue
		}

		value := parts[1]
		if value == "no value" {
			value = ""
		}

		directives[parts[0]] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read phpinfo: %w", err)
	}

	return directives, nil
}

// ReadFpmDirectives runs "php-fpm -i" with the extra arguments of php-fpm and returns the local
// values of the PHP directives. The values come from the php.ini of the FPM SAPI and from the
// "-d" arguments. They do not contain the php_value directives of the pools.
func ReadFpmDirectives(ctx context.Context, fpmPath string, fpmArgs []string) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, fpmPath, append(slices.Clone(fpmArgs), "-i")...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("run %s -i: %w: %s", fpmPath, err, bytes.TrimSpace(stderr.Bytes()))
	}

	return parsePhpinfo(bytes.NewReader(out))
}
