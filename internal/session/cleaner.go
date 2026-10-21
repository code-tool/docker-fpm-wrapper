package session

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/prometheus/procfs"
	"go.uber.org/zap"
)

// MasterProcess is the php-fpm master process.
type MasterProcess interface {
	Pid() int
}

// Cleaner removes the expired session files, as the sessionclean cron job of Debian does.
// It does not remove the files that the php-fpm master or its workers have open: it updates
// their modification time instead.
type Cleaner struct {
	log    *zap.Logger
	master MasterProcess

	targets  []Target
	interval time.Duration
}

func NewCleaner(log *zap.Logger, master MasterProcess, targets []Target, interval time.Duration) *Cleaner {
	return &Cleaner{log: log, master: master, targets: targets, interval: interval}
}

// openFiles returns the files that the php-fpm master and its workers have open.
func (c *Cleaner) openFiles() (map[string]bool, error) {
	procs, err := procfs.AllProcs()
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}

	masterPid := c.master.Pid()
	files := make(map[string]bool)
	for _, proc := range procs {
		if proc.PID != masterPid {
			stat, err := proc.Stat()
			if err != nil || stat.PPID != masterPid {
				continue
			}
		}

		// The process can exit before the read.
		targets, err := proc.FileDescriptorTargets()
		if err != nil {
			continue
		}

		for _, target := range targets {
			files[target] = true
		}
	}

	return files, nil
}

// cleanTarget removes the expired session files in the directory of target and in its
// subdirectories. It returns the number of removed files.
func (c *Cleaner) cleanTarget(ctx context.Context, target Target, openFiles map[string]bool, now time.Time) (int, error) {
	// The kernel gives the paths of the open files without symlinks.
	root, err := filepath.EvalSymlinks(target.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("resolve %s: %w", target.Dir, err)
	}

	removed := 0
	var errs []error
	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, err)
			return nil
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		if !entry.Type().IsRegular() || !strings.HasPrefix(entry.Name(), "sess_") {
			return nil
		}

		if openFiles[path] {
			if err := os.Chtimes(path, now, now); err != nil && !errors.Is(err, fs.ErrNotExist) {
				errs = append(errs, fmt.Errorf("touch open session file: %w", err))
			}

			return nil
		}

		info, err := entry.Info()
		if err != nil {
			// PHP removed the file after the read of the directory.
			return nil
		}

		if now.Sub(info.ModTime()) <= target.MaxLifetime {
			return nil
		}

		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, fmt.Errorf("remove session file: %w", err))
			return nil
		}
		removed++

		return nil
	})

	return removed, errors.Join(walkErr, errors.Join(errs...))
}

// Cleanup removes the session files that are older than the lifetime of their directory at now.
func (c *Cleaner) Cleanup(ctx context.Context, now time.Time) error {
	openFiles, err := c.openFiles()
	if err != nil {
		return err
	}

	var errs []error
	for _, target := range c.targets {
		removed, err := c.cleanTarget(ctx, target, openFiles, now)
		if err != nil {
			errs = append(errs, fmt.Errorf("clean %s: %w", target.Dir, err))
		}

		c.log.Debug("session files removed", zap.String("dir", target.Dir), zap.Int("count", removed))
	}

	return errors.Join(errs...)
}

// Run calls Cleanup at start and then after each interval. It blocks until ctx is done.
func (c *Cleaner) Run(ctx context.Context) error {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	now := time.Now()
	for {
		if err := c.Cleanup(ctx, now); err != nil && ctx.Err() == nil {
			c.log.Error("can't clean up session files", zap.Error(err))
		}

		select {
		case <-ctx.Done():
			return nil
		case now = <-ticker.C:
		}
	}
}
