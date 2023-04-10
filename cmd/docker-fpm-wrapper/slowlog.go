package main

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"golang.org/x/sys/unix"

	"github.com/code-tool/docker-fpm-wrapper/internal/zapx"
	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

func startSlowlogProxyForPool(ctx context.Context, pool phpfpm.Pool, out chan phpfpm.SlowlogEntry) error {
	if err := os.Remove(pool.SlowlogPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("can't remove slowlog file: %w", err)
	}

	if err := unix.Mkfifo(pool.SlowlogPath, 0666); err != nil {
		return fmt.Errorf("can't create linux pipe for slowlog: %w", err)
	}

	fifoF, err := os.OpenFile(pool.SlowlogPath, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return fmt.Errorf("can't open pipe with error: %w", err)
	}

	go func() {
		<-ctx.Done()
		_ = fifoF.Close()
	}()

	slowLogParser := phpfpm.NewSlowlogParser(pool.RequestSlowlogTraceDepth)
	go func() {
		// TODO probably should be logged
		_ = slowLogParser.Parse(ctx, fifoF, out)
	}()

	return nil
}

func startSlowlogProxies(ctx context.Context, fpmConfig phpfpm.Config, log *zap.Logger) error {
	outCh := make(chan phpfpm.SlowlogEntry)
	go func() {
		slowlogEnc := zapx.NewSlowlogEncoder()
		for {
			select {
			case <-ctx.Done():
				return
			case entry := <-outCh:
				if ce := log.Check(zap.WarnLevel, "slowlog detected"); ce != nil {
					ce.Time = entry.CreatedAt
					ce.Write(slowlogEnc.Encode(entry)...)
				}
			}
		}
	}()

	for _, pool := range fpmConfig.Pools {
		if pool.SlowlogPath == "" {
			continue
		}

		if err := startSlowlogProxyForPool(ctx, pool, outCh); err != nil {
			return err
		}
	}

	return nil
}
