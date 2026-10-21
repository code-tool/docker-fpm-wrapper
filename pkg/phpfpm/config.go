package phpfpm

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// defaultSlowlogTraceDepth is the trace depth of php-fpm. PHP before 7.2 has no
// request_slowlog_trace_depth and always uses this depth.
const defaultSlowlogTraceDepth = 20

// Config is the config of php-fpm, as php-fpm reads it.
type Config struct {
	Pools []Pool
}

type Pool struct {
	Name                     string
	Listen                   string
	StatusPath               string
	StatusListen             string
	SlowlogPath              string
	RequestSlowlogTimeout    time.Duration
	RequestSlowlogTraceDepth int
	// PhpValues holds the php_value and php_admin_value directives. php-fpm shows a php_flag
	// directive as php_value.
	PhpValues map[string]string
}

var (
	dumpLineRegexp    = regexp.MustCompile(`^\[[^]]*] NOTICE: (.*)$`)
	dumpSectionRegexp = regexp.MustCompile(`^\[(.+)]$`)
	phpValueKeyRegexp = regexp.MustCompile(`^php(_admin)?_value\[(.+)]$`)
)

// newPool makes a pool from the directives of its section in the dump.
func newPool(name string, directives map[string]string, adminValues map[string]bool) (Pool, error) {
	pool := Pool{
		Name:                     name,
		Listen:                   directives["listen"],
		StatusPath:               directives["pm.status_path"],
		StatusListen:             directives["pm.status_listen"],
		SlowlogPath:              directives["slowlog"],
		RequestSlowlogTraceDepth: defaultSlowlogTraceDepth,
		PhpValues:                make(map[string]string),
	}

	if value := directives["request_slowlog_timeout"]; value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return Pool{}, fmt.Errorf("pool %s: parse request_slowlog_timeout: %w", name, err)
		}
		pool.RequestSlowlogTimeout = timeout
	}

	if value := directives["request_slowlog_trace_depth"]; value != "" {
		depth, err := strconv.Atoi(value)
		if err != nil {
			return Pool{}, fmt.Errorf("pool %s: parse request_slowlog_trace_depth: %w", name, err)
		}
		pool.RequestSlowlogTraceDepth = depth
	}

	for key, value := range directives {
		matches := phpValueKeyRegexp.FindStringSubmatch(key)
		if matches == nil {
			continue
		}

		// An admin directive takes precedence, as in php-fpm.
		phpName, admin := matches[2], matches[1] != ""
		if !admin && adminValues[phpName] {
			continue
		}
		pool.PhpValues[phpName] = value
	}

	return pool, nil
}

// ParseDump reads the output of "php-fpm -tt": the config after php-fpm applied the includes,
// the $pool variables, and the defaults. The global section is "[global]", or "[General]" in
// PHP 5.6. Each other section is a pool.
func ParseDump(r io.Reader) (Config, error) {
	var (
		config      Config
		section     string
		directives  map[string]string
		adminValues map[string]bool
	)

	addPool := func() error {
		if section == "" || section == "global" || section == "General" {
			return nil
		}

		pool, err := newPool(section, directives, adminValues)
		if err != nil {
			return err
		}
		config.Pools = append(config.Pools, pool)

		return nil
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		matches := dumpLineRegexp.FindStringSubmatch(scanner.Text())
		if matches == nil {
			continue
		}
		content := matches[1]

		if sectionMatches := dumpSectionRegexp.FindStringSubmatch(content); sectionMatches != nil {
			if err := addPool(); err != nil {
				return Config{}, err
			}

			section = sectionMatches[1]
			directives = make(map[string]string)
			adminValues = make(map[string]bool)

			continue
		}

		directive, found := strings.CutPrefix(content, "\t")
		if !found || section == "" {
			continue
		}

		key, value, found := strings.Cut(directive, " = ")
		if !found {
			continue
		}
		if value == "undefined" {
			value = ""
		}

		directives[key] = value
		if matches := phpValueKeyRegexp.FindStringSubmatch(key); matches != nil && matches[1] != "" {
			adminValues[matches[2]] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("read php-fpm config dump: %w", err)
	}

	if err := addPool(); err != nil {
		return Config{}, err
	}

	if len(config.Pools) == 0 {
		return Config{}, errors.New("php-fpm config dump has no pools")
	}

	return config, nil
}

// ReadConfig runs "php-fpm -tt" for the config and returns the config that php-fpm reads.
// The extra arguments of php-fpm go before the other arguments, as in Process.
//
// "php-fpm -tt" opens error_log and the slowlog files. It blocks on a FIFO that has no reader,
// so ctx must have a deadline.
func ReadConfig(ctx context.Context, fpmPath, fpmConfigPath string, fpmArgs []string) (Config, error) {
	args := append(slices.Clone(fpmArgs), "--fpm-config", fpmConfigPath, "-tt")
	cmd := exec.CommandContext(ctx, fpmPath, args...)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return Config{}, fmt.Errorf(
				"run php-fpm -tt: no exit before the deadline: a FIFO without a reader at error_log or slowlog can block php-fpm: %w", err)
		}

		if output := bytes.TrimSpace(out.Bytes()); len(output) > 0 {
			return Config{}, fmt.Errorf("run php-fpm -tt: %w: %s", err, output)
		}

		return Config{}, fmt.Errorf("run php-fpm -tt: %w", err)
	}

	return ParseDump(&out)
}
