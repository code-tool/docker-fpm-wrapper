package phpfpm

import (
	"fmt"
	"strings"
	"time"
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

func (se *SlowlogEntry) String() string {
	var (
		err error
		b   strings.Builder
	)

	_, err = fmt.Fprintf(&b, "[%s]  [pool %s] pid %d\n", se.CreatedAt.Format(slowlogTimeFormat), se.PoolName, se.Pid)
	if err != nil {
		return ""
	}

	if _, err = fmt.Fprintf(&b, "script_filename = %s\n", se.ScriptFilename); err != nil {
		return ""
	}

	for i := range se.Stacktrace {
		_, err := fmt.Fprintf(&b, "[%s] %s %s:%d\n",
			se.Stacktrace[i].PtrHex,
			se.Stacktrace[i].FunName,
			se.Stacktrace[i].Path,
			se.Stacktrace[i].Line,
		)

		if err != nil {
			return ""
		}
	}

	return b.String()
}
