package main

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func createFIFOByPath(fPath string) (*os.File, error) {
	if err := os.Remove(fPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("can't remove file: %w", err)
	}

	if err := unix.Mkfifo(fPath, 0666); err != nil {
		return nil, fmt.Errorf("can't create linux pipe: %w", err)
	}

	fifoF, err := os.OpenFile(fPath, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return nil, fmt.Errorf("can't open pipe: %w", err)
	}

	return fifoF, nil
}

func createFIFOByPathCtx(ctx context.Context, fPath string) (*os.File, error) {
	f, err := createFIFOByPath(fPath)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		_ = f.Close()
	}()

	return f, nil
}
