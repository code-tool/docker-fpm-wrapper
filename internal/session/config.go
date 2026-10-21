package session

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// defaultGcMaxLifetime is the default of session.gc_maxlifetime in PHP.
const defaultGcMaxLifetime = 1440 * time.Second

// Config is the session config of one pool.
type Config struct {
	SaveHandler   string
	SavePath      string
	GcMaxLifetime time.Duration
}

// NewConfig reads the session directives. The PHP directives of the pool take precedence over
// the directives of php.ini.
func NewConfig(directives, poolValues map[string]string) (Config, error) {
	get := func(name string) string {
		if value, ok := poolValues[name]; ok {
			return value
		}

		return directives[name]
	}

	config := Config{
		SaveHandler:   get("session.save_handler"),
		SavePath:      get("session.save_path"),
		GcMaxLifetime: defaultGcMaxLifetime,
	}

	if lifetime := get("session.gc_maxlifetime"); lifetime != "" {
		seconds, err := strconv.Atoi(lifetime)
		if err != nil {
			return Config{}, fmt.Errorf("parse session.gc_maxlifetime: %w", err)
		}

		config.GcMaxLifetime = time.Duration(seconds) * time.Second
	}

	return config, nil
}

// Dir returns the directory of the session files. session.save_path can have the form
// "N;/path" or "N;MODE;/path", where N is the number of subdirectory levels.
func (c Config) Dir() string {
	if _, dir, found := strings.CutLast(c.SavePath, ";"); found {
		return dir
	}

	return c.SavePath
}

// HasFiles tells if PHP keeps the sessions as files in a known directory.
func (c Config) HasFiles() bool {
	return c.SaveHandler == "files" && c.Dir() != ""
}
