package phpfpm

import (
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

type Config struct {
	Include  string
	ErrorLog string
	Pools    []Pool
}

type Pool struct {
	Name                     string
	Listen                   string
	StatusPath               string
	StatusListen             string
	SlowlogPath              string
	RequestSlowlogTimeout    int
	RequestSlowlogTraceDepth int
	// PhpValues holds the php_value, php_flag, php_admin_value, and php_admin_flag directives.
	PhpValues map[string]string
}

var phpValueKeyRegexp = regexp.MustCompile(`^php(_admin)?_(?:value|flag)\[(.+)]$`)

// parsePhpValues returns the PHP directives of the pool section with expand applied to the
// values. An admin directive takes precedence over a directive with the same name, as in php-fpm.
func parsePhpValues(section *ini.Section, expand func(string) string) map[string]string {
	values := make(map[string]string)
	isAdmin := make(map[string]bool)

	for _, key := range section.Keys() {
		matches := phpValueKeyRegexp.FindStringSubmatch(key.Name())
		if matches == nil {
			continue
		}

		name, admin := matches[2], matches[1] != ""
		if isAdmin[name] && !admin {
			continue
		}

		values[name] = expand(key.String())
		isAdmin[name] = admin
	}

	return values
}

func fillPull(config *Config, iniConfig *ini.File, poolName string) error {
	pool := Pool{}
	pool.Name = poolName

	section, err := iniConfig.GetSection(poolName)
	if err != nil {
		return err
	}

	// php-fpm replaces each $pool in each directive with the name of the pool.
	expand := func(value string) string {
		return strings.ReplaceAll(value, "$pool", poolName)
	}
	get := func(name string) (string, bool) {
		key, err := section.GetKey(name)
		if err != nil {
			return "", false
		}

		return expand(key.String()), true
	}

	listen, ok := get("listen")
	if !ok {
		return errors.New(`key "listen" not exists`)
	}
	pool.Listen = listen

	pool.StatusPath, _ = get("pm.status_path")
	pool.StatusListen, _ = get("pm.status_listen")
	pool.SlowlogPath, _ = get("slowlog")

	if value, ok := get("request_slowlog_timeout"); ok {
		pool.RequestSlowlogTimeout, _ = strconv.Atoi(value)
	}

	pool.RequestSlowlogTraceDepth = 64
	if value, ok := get("request_slowlog_trace_depth"); ok {
		pool.RequestSlowlogTraceDepth, _ = strconv.Atoi(value)
	}

	pool.PhpValues = parsePhpValues(section, expand)

	config.Pools = append(config.Pools, pool)
	return nil
}

func ParseConfig(fpmConfigPath string) (Config, error) {
	c := Config{}
	// A ";" after a space starts a comment. A ";" in a value stays, for example "2;/var/lib/php/sessions".
	cfg := ini.Empty(ini.LoadOptions{SpaceBeforeInlineComment: true})
	err := cfg.Append(fpmConfigPath)
	if err != nil {
		return c, err
	}

	global, err := cfg.GetSection("global")
	if err != nil {
		return c, err
	}

	if key, err := global.GetKey("error_log"); err == nil {
		c.ErrorLog = key.String()
	}

	if include, err := global.GetKey("include"); err == nil {
		c.Include = include.Value()

		file := regexp.QuoteMeta(path.Base(c.Include))
		file = strings.Replace(file, regexp.QuoteMeta("*"), "(.+)", 1)
		file = fmt.Sprintf("^%s$", file)
		fileRx := regexp.MustCompile(file)

		fmpPoolsDir := path.Dir(c.Include)
		osFileInfo, err := os.ReadDir(fmpPoolsDir)
		if err != nil {
			return c, err
		}

		for _, info := range osFileInfo {
			if info.IsDir() || !fileRx.MatchString(info.Name()) {
				continue
			}

			if err = cfg.Append(fmt.Sprintf("%s/%s", fmpPoolsDir, info.Name())); err != nil {
				return c, err
			}
		}
	}

	// php-fpm takes each section except [global] as a pool.
	for _, section := range cfg.Sections() {
		if section.Name() == ini.DefaultSection || section.Name() == "global" {
			continue
		}

		if err := fillPull(&c, cfg, section.Name()); err != nil {
			return c, fmt.Errorf("pool %s: %w", section.Name(), err)
		}
	}

	return c, nil
}
