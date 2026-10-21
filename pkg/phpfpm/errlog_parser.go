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
	unparsedOut io.Writer
}

// NewErrLogParser returns a parser of the php-fpm error log. The parser writes the lines that
// are not in the php-fpm format to unparsedOut without a change. Such lines come, for example,
// from a pool with "decorate_workers_output = no".
func NewErrLogParser(unparsedOut io.Writer) *ErrLogParser {
	return &ErrLogParser{unparsedOut: unparsedOut}
}

var errLogEntryRegexp = regexp.MustCompile(`^\[([^]]+)]\s+(ALERT|ERROR|WARNING|NOTICE|DEBUG):\s+([^\n]+)\n$`)

func (p *ErrLogParser) parseLine(buf []byte) (ErrLogEntry, error) {
	var err error
	result := ErrLogEntry{}

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

func (p *ErrLogParser) ParseOne(r *bufio.Reader) (ErrLogEntry, error) {
	buf, err := line.ReadOne(r, false)
	if err != nil {
		return ErrLogEntry{}, err
	}

	return p.parseLine(buf)
}

func (p *ErrLogParser) Parse(ctx context.Context, r io.Reader, out chan ErrLogEntry) error {
	bufioReader := bufio.NewReader(r)

	for {
		buf, err := line.ReadOne(bufioReader, false)
		if err != nil {
			return err
		}

		entry, err := p.parseLine(buf)
		if err != nil {
			if _, err := p.unparsedOut.Write(buf); err != nil {
				return fmt.Errorf("write unparsed line: %w", err)
			}

			continue
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			out <- entry
		}
	}
}
