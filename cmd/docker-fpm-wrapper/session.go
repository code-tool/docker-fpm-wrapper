package main

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/code-tool/docker-fpm-wrapper/internal/session"
	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

// createSessionCleaner reads the session config of php-fpm and returns a cleaner for its
// session directories. It returns nil when no pool keeps the sessions as files.
func createSessionCleaner(
	ctx context.Context,
	log *zap.Logger,
	fpmProcess *phpfpm.Process,
	fpmPath string, fpmArgs []string,
	pools []phpfpm.Pool,
	interval time.Duration,
) (*session.Cleaner, error) {
	directives, err := session.ReadFpmDirectives(ctx, fpmPath, fpmArgs)
	if err != nil {
		return nil, fmt.Errorf("read php-fpm directives: %w", err)
	}

	targets, err := session.NewTargets(directives, pools)
	if err != nil {
		return nil, fmt.Errorf("find session directories: %w", err)
	}

	if len(targets) == 0 {
		log.Warn("no session directories to clean up")
		return nil, nil
	}

	for _, target := range targets {
		log.Info("session cleanup enabled",
			zap.String("dir", target.Dir), zap.Duration("max_lifetime", target.MaxLifetime), zap.Duration("interval", interval))
	}

	return session.NewCleaner(log, fpmProcess, targets, interval), nil
}
