package main

import (
	"context"

	"go.uber.org/zap"

	"github.com/code-tool/docker-fpm-wrapper/internal/zapx"
	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

func startSlowlogProxyForPool(ctx context.Context, pool phpfpm.Pool, out chan phpfpm.SlowlogEntry) error {
	fifoF, err := createFIFOByPathCtx(ctx, pool.SlowlogPath)
	if err != nil {
		return err
	}

	slowLogParser := phpfpm.NewSlowlogParser(pool.RequestSlowlogTraceDepth)
	go func() {
		// TODO probably should be logged
		_ = slowLogParser.Parse(ctx, fifoF, out)
	}()

	return nil
}

func startSlowlogProxies(ctx context.Context, log *zap.Logger, pools []phpfpm.Pool) error {
	outCh := make(chan phpfpm.SlowlogEntry)
	go func() {
		slowlogEnc := zapx.NewSlowlogEncoder()
		for {
			select {
			case <-ctx.Done():
				return
			case entry := <-outCh:
				if ce := log.Check(zap.WarnLevel, "slowlog"); ce != nil {
					ce.Time = entry.CreatedAt
					ce.Write(slowlogEnc.Encode(entry)...)
				}
			}
		}
	}()

	for _, pool := range pools {
		if pool.SlowlogPath == "" {
			continue
		}

		if err := startSlowlogProxyForPool(ctx, pool, outCh); err != nil {
			return err
		}
	}

	return nil
}
