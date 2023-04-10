package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"sort"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sys/unix"

	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

type slowlogToZapEncoder struct {
	strBuf []string
}

func (sle *slowlogToZapEncoder) reset() {
	sle.strBuf = sle.strBuf[:0]
}

func (sle *slowlogToZapEncoder) add(s string) {
	if sle.strBuf[len(sle.strBuf)-1] == s {
		return
	}

	i := sort.SearchStrings(sle.strBuf, s)
	sle.strBuf = append(sle.strBuf, "")
	copy(sle.strBuf[i+1:], sle.strBuf[i:])

	sle.strBuf[i] = s
}

func (sle *slowlogToZapEncoder) addDir(p string) {
	sle.add(path.Dir(p))
}

func (sle *slowlogToZapEncoder) longestCommonPrefOffset() int {
	offset := 0
	endPrefix := false

	if len(sle.strBuf) <= 0 {
		return 0
	}

	first := sle.strBuf[0]
	last := sle.strBuf[len(sle.strBuf)-1]

	for i := 0; i < len(first); i++ {
		if !endPrefix && string(last[i]) == string(first[i]) {
			offset = i + 1
		} else {
			endPrefix = true
		}
	}

	return offset
}

func (sle *slowlogToZapEncoder) encodeStacktraceEntry(encoder zapcore.ObjectEncoder, entry phpfpm.SlowlogTraceEntry, pathOffset int) {
	encoder.AddString("path", entry.Path[pathOffset:])
	encoder.AddString("func", entry.FunName)
	encoder.AddInt("line", entry.Line)
}

func (sle *slowlogToZapEncoder) encodeStacktrace(stacktrace []phpfpm.SlowlogTraceEntry, pathOffset int) zap.Field {
	return zap.Array("trace", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for i := range stacktrace {
			err := encoder.AppendObject(
				zapcore.ObjectMarshalerFunc(func(encoder zapcore.ObjectEncoder) error {
					sle.encodeStacktraceEntry(encoder, stacktrace[i], pathOffset)

					return nil
				}),
			)

			if err != nil {
				return err
			}
		}

		return nil
	}))
}

func (sle *slowlogToZapEncoder) Encode(entry phpfpm.SlowlogEntry) []zap.Field {
	sle.reset()

	sle.addDir(entry.ScriptFilename)
	for i := range entry.Stacktrace {
		sle.addDir(entry.Stacktrace[i].Path)
	}

	pathOffset := sle.longestCommonPrefOffset()

	return []zap.Field{
		zap.String("filename", entry.ScriptFilename[pathOffset:]),
		sle.encodeStacktrace(entry.Stacktrace, pathOffset),
	}
}

func startSlowlogProxyForPool(ctx context.Context, pool phpfpm.Pool, out chan phpfpm.SlowlogEntry) error {
	if err := os.Remove(pool.SlowlogPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := unix.Mkfifo(pool.SlowlogPath, 0666); err != nil {
		return err
	}

	fifoF, err := os.OpenFile(pool.SlowlogPath, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		fmt.Println("Couldn't open pipe with error: ", err)
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
		slowlogEnc := &slowlogToZapEncoder{strBuf: make([]string, 0, 16)}
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
