package main

import (
	"context"

	"go.uber.org/zap"

	"github.com/code-tool/docker-fpm-wrapper/internal/zapx"
	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

func startSlowlogProxy(ctx context.Context, log *zap.Logger, fPath string) error {
	if fPath == "" {
		return nil
	}

	f, err := createFIFOByPathCtx(ctx, fPath)
	if err != nil {
		return err
	}

	entryCh := make(chan phpfpm.ErrLogEntry)
	go func() {
		for {
			select {
			case entry := <-entryCh:
				if ce := log.Check(zapx.MapFpmLogLevel(entry.Level), entry.Message); ce != nil {
					ce.Time = entry.CreatedAt
					ce.Write()
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	logParser := phpfpm.NewErrLogParser()
	go func() {
		_ = logParser.Parse(ctx, f, entryCh)
	}()

	return nil
}
