package zapx

import (
	"fmt"
	"strconv"

	"go.uber.org/zap/zapcore"
)

// LogLevel is a zap level that takes a name ("debug") or a number ("-1") from a flag.
// It implements flags.Unmarshaler of go-flags. Its kind is int8, so go-flags accepts a negative
// number as the argument of a flag.
type LogLevel zapcore.Level

// UnmarshalFlag implements flags.Unmarshaler.
func (l *LogLevel) UnmarshalFlag(value string) error {
	if level, err := zapcore.ParseLevel(value); err == nil {
		*l = LogLevel(level)
		return nil
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("unknown log level %q", value)
	}

	level := zapcore.Level(number)
	if level < zapcore.DebugLevel || level > zapcore.FatalLevel {
		return fmt.Errorf("log level %d is out of range [%d, %d]", number, zapcore.DebugLevel, zapcore.FatalLevel)
	}

	*l = LogLevel(level)

	return nil
}

func (l LogLevel) Level() zapcore.Level {
	return zapcore.Level(l)
}
