package zapx

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

func MapFpmLogLevel(ll phpfpm.LogLevel) zapcore.Level {
	switch ll {
	case phpfpm.LogLevelAlert:
		return zap.FatalLevel
	case phpfpm.LogLevelError:
		return zap.ErrorLevel
	case phpfpm.LogLevelWarning:
		return zap.WarnLevel
	case phpfpm.LogLevelNotice:
		return zap.InfoLevel
	case phpfpm.LogLevelDebug:
		return zap.DebugLevel
	default:
		return zap.DebugLevel
	}
}
