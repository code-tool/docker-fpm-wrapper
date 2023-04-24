package phpfpm

import (
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
}

func fillPull(config *Config, iniConfig *ini.File, poolName string) error {
	pool := Pool{}
	pool.Name = poolName

	section, err := iniConfig.GetSection(poolName)
	if err != nil {
		return err
	}

	key, err := section.GetKey("listen")
	if err != nil {
		return err
	}
	pool.Listen = key.String()

	key, err = section.GetKey("pm.status_path")
	if err == nil {
		pool.StatusPath = key.String()
	}

	key, err = section.GetKey("pm.status_listen")
	if err == nil {
		pool.StatusListen = key.String()
	}

	key, err = section.GetKey("slowlog")
	if err == nil {
		pool.SlowlogPath = strings.Replace(key.String(), "$pool", poolName, 1)
	}

	key, err = section.GetKey("request_slowlog_timeout")
	if err == nil {
		pool.RequestSlowlogTimeout, _ = strconv.Atoi(key.String())
	}

	pool.RequestSlowlogTraceDepth = 20
	key, err = section.GetKey("request_slowlog_trace_depth")
	if err == nil {
		pool.RequestSlowlogTraceDepth, _ = strconv.Atoi(key.String())
	}

	config.Pools = append(config.Pools, pool)
	return nil
}

func ParseConfig(fpmConfigPath string) (Config, error) {
	c := Config{}
	cfg := ini.Empty()
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

	for _, section := range cfg.Sections() {
		_, err = section.GetKey("pm.status_path")
		if err != nil {
			continue
		}

		if err := fillPull(&c, cfg, section.Name()); err != nil {
			return c, err
		}
	}

	return c, err
}
