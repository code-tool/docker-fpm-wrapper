package session

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

// Target is a directory with session files and the lifetime of the files in it.
type Target struct {
	Dir         string
	MaxLifetime time.Duration
}

// NewTargets returns the session directories of the pools, sorted by directory. It skips the
// pools that do not keep the sessions as files. When two or more pools use one directory, the
// longest lifetime applies. Then the cleaner does not remove a live session of a pool.
func NewTargets(directives map[string]string, pools []phpfpm.Pool) ([]Target, error) {
	lifetimes := make(map[string]time.Duration)

	for _, pool := range pools {
		config, err := NewConfig(directives, pool.PhpValues)
		if err != nil {
			return nil, fmt.Errorf("pool %s: %w", pool.Name, err)
		}

		if !config.HasFiles() {
			continue
		}

		dir := filepath.Clean(config.Dir())
		lifetimes[dir] = max(lifetimes[dir], config.GcMaxLifetime)
	}

	targets := make([]Target, 0, len(lifetimes))
	for dir, lifetime := range lifetimes {
		targets = append(targets, Target{Dir: dir, MaxLifetime: lifetime})
	}

	slices.SortFunc(targets, func(a, b Target) int {
		return cmp.Compare(a.Dir, b.Dir)
	})

	return targets, nil
}
