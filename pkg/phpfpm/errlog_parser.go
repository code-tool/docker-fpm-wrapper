package phpfpm

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/code-tool/docker-fpm-wrapper/pkg/line"
)

type ErrLogEntry struct {
	CreatedAt time.Time
	Level     LogLevel
	Message   string
}

type ErrLogParser struct {
	//
}

func NewErrLogParser() *ErrLogParser {
	return &ErrLogParser{}
}

var errLogEntryRegexp = regexp.MustCompile(`^\[([^]]+)]\s+(ALERT|ERROR|WARNING|NOTICE|DEBUG):\s+([^\n]+)\n$`)

func (p *ErrLogParser) ParseOne(r *bufio.Reader) (ErrLogEntry, error) {
	result := ErrLogEntry{}
	buf, err := line.ReadOne(r)
	if err != nil {
		return result, err
	}

	matches := errLogEntryRegexp.FindSubmatchIndex(buf)
	if len(matches) == 0 {
		return result, errors.New("unexpected log line format")
	}
	//
	result.CreatedAt, err = time.Parse("02-Jan-2006 15:04:05", string(buf[matches[2]:matches[3]]))
	if err != nil {
		return result, fmt.Errorf("can't parse timestamp: %w", err)
	}

	result.Level = LogLevel(buf[matches[4]:matches[5]])
	result.Message = string(buf[matches[6]:matches[7]])

	return result, nil
}

func (p *ErrLogParser) Parse(ctx context.Context, r io.Reader, out chan ErrLogEntry) error {
	bufioReader := bufio.NewReader(r)

	for {
		entry, err := p.ParseOne(bufioReader)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			out <- entry
		}
	}
}
