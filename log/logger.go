package log

var logger Logger

type Logger interface {
	LogDebug(format string, args ...interface{})
	LogInfo(format string, args ...interface{})
	LogWarn(format string, args ...interface{})
	LogError(format string, args ...interface{})
	LogFatal(format string, args ...interface{})
}

func SetLogger(log Logger) {
	logger = log
}

func Debug(format string, args ...interface{}) {
	logger.LogDebug(format, args...)
}

func Info(format string, args ...interface{}) {
	logger.LogInfo(format, args...)
}

func Warn(format string, args ...interface{}) {
	logger.LogWarn(format, args...)
}

func Error(format string, args ...interface{}) {
	logger.LogError(format, args...)
}

func Fatal(format string, args ...interface{}) {
	logger.LogFatal(format, args...)
}
