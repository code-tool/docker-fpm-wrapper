package phpfpm

type LogLevel string

const (
	LogLevelAlert   = "ALERT"
	LogLevelError   = "ERROR"
	LogLevelWarning = "WARNING"
	LogLevelNotice  = "NOTICE"
	LogLevelDebug   = "DEBUG"
)

const logTimeFormat = "02-Jan-2006 15:04:05"
