package phpfpm

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/code-tool/docker-fpm-wrapper/pkg/line"
)

const (
	stateParseHeader = iota
	stateParseFilename
	stateParseStacktrace
)

const stacktraceParsingTimeout = 25 * time.Millisecond
const slowlogTimeFormat = "02-Jan-2006 15:04:05"

var (
	headerRegexp          = regexp.MustCompile(`^\[([^]]+)]\s+\[pool\s([^]]+)]\s+pid\s+(\d+)(?:[\r\n]|$)`)
	stacktraceEntryRegexp = regexp.MustCompile(`^\[([^]]+)]\s+(\S+)\s+([^:]+):(\d+)(?:[\r\n]+|$)`)
)

type SlowlogParser struct {
	maxTraceLen int
}

func NewSlowlogParser(maxTraceLen int) *SlowlogParser {
	return &SlowlogParser{maxTraceLen: maxTraceLen}
}

func (slp *SlowlogParser) parseHeader(line []byte, entry *SlowlogEntry) error {
	var err error
	matches := headerRegexp.FindSubmatchIndex(line)
	if len(matches) == 0 {
		return errors.New("not a header")
	}

	entry.CreatedAt, err = time.Parse("02-Jan-2006 15:04:05", string(line[matches[2]:matches[3]]))
	if err != nil {
		return err
	}
	entry.PoolName = string(line[matches[4]:matches[5]])

	pidStr := string(line[matches[6]:matches[7]])
	if entry.Pid, err = strconv.Atoi(pidStr); err != nil {
		return err
	}

	return nil
}

func (slp *SlowlogParser) parseFilename(line []byte, entry *SlowlogEntry) error {
	prefix := []byte("script_filename = ")
	if !bytes.HasPrefix(line, prefix) {
		return errors.New("not filename line")
	}
	entry.ScriptFilename = string(line[len(prefix) : len(line)-1])

	return nil
}

func (slp *SlowlogParser) parseStacktraceEntry(line []byte, entry *SlowlogEntry) error {
	matches := stacktraceEntryRegexp.FindSubmatchIndex(line)
	if len(matches) == 0 {
		return errors.New("not a stacktrace entry")
	}

	lineN, err := strconv.Atoi(string(line[matches[8]:matches[9]]))
	if err != nil {
		return err
	}

	entry.Stacktrace = append(entry.Stacktrace, SlowlogTraceEntry{
		PtrHex:  string(line[matches[2]:matches[3]]),
		FunName: string(line[matches[4]:matches[5]]),
		Path:    string(line[matches[6]:matches[7]]),
		Line:    lineN,
	})

	return nil
}

func (slp *SlowlogParser) parseLine(line []byte, entry *SlowlogEntry, state *int) bool {
	switch *state {
	case stateParseHeader:
		if err := slp.parseHeader(line, entry); err != nil {
			entry.Reset()
			break
		}
		*state = stateParseFilename
	case stateParseFilename:
		if err := slp.parseFilename(line, entry); err != nil {
			entry.Reset()
			*state = stateParseHeader
			break
		}
		*state = stateParseStacktrace
	case stateParseStacktrace:
		if bytes.Equal(line, []byte{'\n'}) {
			*state = stateParseHeader
			return true
		}

		if err := slp.parseStacktraceEntry(line, entry); err != nil {
			entry.Reset()
			*state = stateParseHeader
			break
		}

		if slp.maxTraceLen > 0 && len(entry.Stacktrace) >= slp.maxTraceLen {
			return true
		}
	default:
		panic("unexpected state")
	}

	return false
}

func (slp *SlowlogParser) createEntry() SlowlogEntry {
	return SlowlogEntry{Stacktrace: make([]SlowlogTraceEntry, 0, slp.maxTraceLen)}
}

func (slp *SlowlogParser) Parse(ctx context.Context, r io.Reader, out chan SlowlogEntry) error {
	errCh := make(chan error)
	lineCh := make(chan []byte)

	go func() {
		bufioReader := bufio.NewReader(r)

		for {
			buf, err := line.ReadOne(bufioReader)
			if err != nil {
				errCh <- err

				return
			}

			select {
			case <-ctx.Done():
				return
			default:
				lineCopy := make([]byte, len(buf))
				copy(lineCopy, buf)

				lineCh <- lineCopy
			}
		}
	}()

	timeoutTimer := time.NewTimer(25 * time.Millisecond)
	timeoutTimer.Stop()

	entry := slp.createEntry()
	state := stateParseHeader

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			timeoutTimer.Stop()
			return err
		case <-timeoutTimer.C:
			out <- entry
			entry = slp.createEntry()
			state = stateParseHeader
		case line := <-lineCh:
			if slp.parseLine(line, &entry, &state) {
				timeoutTimer.Stop()
				out <- entry
				entry = slp.createEntry()
				continue
			}

			if state == stateParseStacktrace {
				timeoutTimer.Reset(stacktraceParsingTimeout)
			}
		}
	}
}
