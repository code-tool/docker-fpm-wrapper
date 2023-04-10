package main

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newZapEncoderConfig() zapcore.EncoderConfig {
	result := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "channel",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	return result
}

func getZapEncoding() string {
	if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return "console"
	}

	return "json"
}

func createLoggerEncoder(eName string, encoderConfig zapcore.EncoderConfig) (zapcore.Encoder, error) {
	if eName == "auto" {
		eName = getZapEncoding()
	}

	switch eName {
	case "console":
		return zapcore.NewConsoleEncoder(encoderConfig), nil
	case "json":
		return zapcore.NewJSONEncoder(encoderConfig), nil
	default:
		return nil, fmt.Errorf("unknown encoder: %s", eName)
	}
}

func createLogger(encName string, level int, output zapcore.WriteSyncer) (*zap.Logger, error) {
	enc, err := createLoggerEncoder(encName, newZapEncoderConfig())
	if err != nil {
		return nil, err
	}

	atomicLevel := zap.NewAtomicLevelAt(zapcore.Level(level))

	return zap.New(zapcore.NewCore(enc, output, atomicLevel)), nil
}
