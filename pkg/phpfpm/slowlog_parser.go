package phpfpm

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"regexp"
	"strconv"
	"time"
)

const (
	stateParseHeader = iota
	stateParseFilename
	stateParseStacktrace
)

var (
	headerRegexp          = regexp.MustCompile(`^\[([^]]+)]\s+\[pool\s([^]]+)]\s+pid\s+(\d+)(?:[\r\n]|$)`)
	stacktraceEntryRegexp = regexp.MustCompile(`^\[([^]]+)]\s+(\S+)\s+([^:]+):(\d+)(?:[\r\n]+|$)`)
)

type SlowlogTraceEntry struct {
	PtrHex  string
	FunName string
	Path    string
	Line    int
}

type SlowlogEntry struct {
	CreatedAt      time.Time
	PoolName       string
	Pid            int
	ScriptFilename string
	Stacktrace     []SlowlogTraceEntry
}

func (se *SlowlogEntry) Reset() {
	se.CreatedAt = time.Time{}
	se.PoolName = ""
	se.Pid = 0
	se.ScriptFilename = ""
	se.Stacktrace = se.Stacktrace[:0]
}

type SlowlogParser struct {
}

func NewSlowlogParser() *SlowlogParser {
	return &SlowlogParser{}
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

func (slp *SlowlogParser) parseSingleEntry(bufioReader *bufio.Reader) (SlowlogEntry, error) {
	skip := false
	state := stateParseHeader

	entry := SlowlogEntry{
		Stacktrace: make([]SlowlogTraceEntry, 0),
	}

	for {
		line, err := bufioReader.ReadSlice('\n')
		if errors.Is(err, io.EOF) {
			return entry, err
		}

		if errors.Is(err, bufio.ErrBufferFull) {
			skip = true
			continue
		}

		if err != nil {
			return entry, err
		}

		if skip {
			skip = false
			continue
		}

		switch state {
		case stateParseHeader:
			if err := slp.parseHeader(line, &entry); err != nil {
				entry.Reset()
				continue
			}
			state = stateParseFilename
		case stateParseFilename:
			if err := slp.parseFilename(line, &entry); err != nil {
				entry.Reset()
				state = stateParseFilename
				continue
			}
			state = stateParseStacktrace
		case stateParseStacktrace:
			if bytes.Compare(line, []byte{'\n'}) == 0 {
				return entry, nil
			}
			if err := slp.parseStacktraceEntry(line, &entry); err != nil {
				entry.Reset()
				state = stateParseHeader
			}
		}
	}
}

func (slp *SlowlogParser) Parse(r io.Reader, out chan SlowlogEntry) error {
	bufioReader := bufio.NewReader(r)

	for {
		entry, err := slp.parseSingleEntry(bufioReader)
		if err != nil {
			return err
		}

		out <- entry
	}
}
