package phpfpm

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strings"

	"gopkg.in/ini.v1"
)

type Config struct {
	Include string
	Pools   []Pool
}

type Pool struct {
	Name       string
	Listen     string
	StatusPath string
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

	include, err := global.GetKey("include")
	if err != nil {
		return c, err
	}

	c.Include = include.Value()

	file := regexp.QuoteMeta(path.Base(c.Include))
	file = strings.Replace(file, regexp.QuoteMeta("*"), "(.+)", 1)
	file = fmt.Sprintf("^%s$", file)
	fileRx := regexp.MustCompile(file)

	fmpPoolsDir := path.Dir(c.Include)
	osFileInfo, err := ioutil.ReadDir(fmpPoolsDir)
	if err != nil {
		return c, err
	}

	for _, info := range osFileInfo {
		if info.IsDir() || !fileRx.MatchString(info.Name()) {
			continue
		}

		err = cfg.Append(fmt.Sprintf("%s/%s", fmpPoolsDir, info.Name()))
		if err != nil {
			return c, err
		}
	}

	for _, section := range cfg.Sections() {
		_, err = section.GetKey("pm.status_path")
		if err != nil {
			continue
		}

		fillPull(&c, cfg, section.Name())
	}

	return c, err
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

	config.Pools = append(config.Pools, pool)
	return nil
}
