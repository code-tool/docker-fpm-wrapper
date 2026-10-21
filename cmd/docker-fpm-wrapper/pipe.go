package main

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

const (
	// The php-fpm workers write to the wrapper pipe. They can run as a different user than the wrapper.
	wrapperPipePerm os.FileMode = 0o666
	// Only the php-fpm master writes to the error log and the slowlog.
	fpmLogPipePerm os.FileMode = 0o644
)

// createFIFOByPath creates a FIFO with perm. The umask of the process does not change perm.
func createFIFOByPath(fPath string, perm os.FileMode) (*os.File, error) {
	if err := os.Remove(fPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("can't remove file: %w", err)
	}

	if err := unix.Mkfifo(fPath, uint32(perm)); err != nil {
		return nil, fmt.Errorf("can't create linux pipe: %w", err)
	}

	if err := os.Chmod(fPath, perm); err != nil {
		return nil, fmt.Errorf("can't change pipe mode: %w", err)
	}

	fifoF, err := os.OpenFile(fPath, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return nil, fmt.Errorf("can't open pipe: %w", err)
	}

	return fifoF, nil
}

func createFIFOByPathCtx(ctx context.Context, fPath string, perm os.FileMode) (*os.File, error) {
	f, err := createFIFOByPath(fPath, perm)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		_ = f.Close()
	}()

	return f, nil
}
