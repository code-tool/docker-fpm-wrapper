package main

import (
	"context"
	"io"

	"go.uber.org/zap"

	"github.com/code-tool/docker-fpm-wrapper/internal/zapx"
	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

func startErrLogProxy(ctx context.Context, log *zap.Logger, fPath string) error {
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
		if err := logParser.Parse(ctx, f, entryCh); err != nil {
			log.Error("can't parse php-fpm errorlog entry", zap.Error(err))
		}
		_, _ = io.Copy(io.Discard, f)
	}()

	return nil
}
