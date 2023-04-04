package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

func startSlowlogProxyForPool(ctx context.Context, pool phpfpm.Pool, out chan phpfpm.SlowlogEntry) error {
	if err := syscall.Mkfifo(pool.SlowlogPath, 0640); err != nil {
		return err
	}

	fifoF, err := os.OpenFile(pool.SlowlogPath, os.O_RDONLY, 0640)
	if err != nil {
		fmt.Println("Couldn't open pipe with error: ", err)
	}

	slowLogParser := phpfpm.NewSlowlogParser()

	go func() {
		<-ctx.Done()
		_ = fifoF.Close()
	}()

	go func() {
		// TODO probably should be logged
		_ = slowLogParser.Parse(fifoF, out)
	}()

	return nil
}

func startSlowlogProxies(ctx context.Context, fpmConfig phpfpm.Config, w io.Writer) error {
	outCh := make(chan phpfpm.SlowlogEntry)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case entry := <-outCh:
				_, _ = w.Write([]byte(entry.String()))
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
