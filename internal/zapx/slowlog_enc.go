package zapx

import (
	"path"
	"sort"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

type SlowlogEncoder struct {
	strBuf []string
}

func NewSlowlogEncoder() *SlowlogEncoder {
	return &SlowlogEncoder{strBuf: make([]string, 0)}
}

func (sle *SlowlogEncoder) reset() {
	sle.strBuf = sle.strBuf[:0]
}

func (sle *SlowlogEncoder) add(s string) {
	if l := len(sle.strBuf); l > 1 && sle.strBuf[l-1] == s {
		return
	}

	i := sort.SearchStrings(sle.strBuf, s)
	sle.strBuf = append(sle.strBuf, "")
	copy(sle.strBuf[i+1:], sle.strBuf[i:])

	sle.strBuf[i] = s
}

func (sle *SlowlogEncoder) addDir(p string) {
	sle.add(path.Dir(p))
}

func (sle *SlowlogEncoder) longestCommonPrefOffset() int {
	offset := 0
	endPrefix := false

	if len(sle.strBuf) <= 0 {
		return 0
	}

	first := sle.strBuf[0]
	last := sle.strBuf[len(sle.strBuf)-1]

	for i := 0; i < len(first); i++ {
		if !endPrefix && last[i] == first[i] {
			offset = i + 1
		} else {
			endPrefix = true
		}
	}

	return offset
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
