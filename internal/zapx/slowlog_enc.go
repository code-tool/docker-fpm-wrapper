package zapx

import (
	"path"
	"slices"
	"sort"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

type SlowlogEncoder struct {
	strBuf []string
}

func NewSlowlogEncoder() *SlowlogEncoder {
	return &SlowlogEncoder{strBuf: make([]string, 0, 32)}
}

func (sle *SlowlogEncoder) reset() {
	sle.strBuf = sle.strBuf[:0]
}

func (sle *SlowlogEncoder) add(s string) {
	i := sort.SearchStrings(sle.strBuf, s)
	if i < len(sle.strBuf) && sle.strBuf[i] == s {
		// Do not insert duplicates
		return
	}

	sle.strBuf = slices.Insert(sle.strBuf, i, s)
}

func (sle *SlowlogEncoder) addDir(p string) bool {
	dir, _ := path.Split(p)
	if dir == "" {
		return false
	}

	sle.add(path.Clean(dir))

	return true
}

func (sle *SlowlogEncoder) longestCommonPrefOffset() int {
	if len(sle.strBuf) <= 0 {
		return 0
	}

	if len(sle.strBuf) == 1 {
		return len(sle.strBuf[0]) + 1
	}

	first := sle.strBuf[0]
	last := sle.strBuf[len(sle.strBuf)-1]

	for i := 0; i < len(first); i++ {
		if last[i] != first[i] {
			return i
		}
	}

	return 0
}

func (sle *SlowlogEncoder) encodeStacktraceEntry(encoder zapcore.ObjectEncoder, entry phpfpm.SlowlogTraceEntry, pathOffset int) {
	encoder.AddString("path", entry.Path[pathOffset:])
	encoder.AddString("func", entry.FunName)
	encoder.AddInt("line", entry.Line)
}

func (sle *SlowlogEncoder) encodeStacktrace(stacktrace []phpfpm.SlowlogTraceEntry, pathOffset int) zap.Field {
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

func (sle *SlowlogEncoder) Encode(entry phpfpm.SlowlogEntry) []zap.Field {
	sle.reset()

	cutPrefix := sle.addDir(entry.ScriptFilename)
	for i := range entry.Stacktrace {
		cutPrefix = cutPrefix && sle.addDir(entry.Stacktrace[i].Path)
	}

	pathOffset := 0
	if cutPrefix {
		pathOffset = sle.longestCommonPrefOffset()
	}

	return []zap.Field{
		zap.String("filename", entry.ScriptFilename[pathOffset:]),
		sle.encodeStacktrace(entry.Stacktrace, pathOffset),
	}
}
