package applog

import (
	"bufio"
	"errors"
	"io"

	"go.uber.org/zap"

	"github.com/code-tool/docker-fpm-wrapper/pkg/line"
)

type PipeProxy struct {
	log    *zap.Logger
	writer io.Writer
}

func NewPipeProxy(log *zap.Logger, writer io.Writer) *PipeProxy {
	return &PipeProxy{log: log, writer: writer}
}

func (p *PipeProxy) Proxy(r io.Reader) {
	bufioReader := bufio.NewReader(r)

	for {
		buf, err := line.ReadOne(bufioReader, true)
		if len(buf) > 0 {
			_, _ = p.writer.Write(normalizeLine(buf))
		}

		if err == nil {
			continue
		}

		if errors.Is(err, io.EOF) {
			return
		}

		p.log.Error("can't read line from pipe", zap.Error(err))
	}
}
