package log

import "github.com/gzjjyz/logger"

var Logger logger.ILogger

func SetLogger(l logger.ILogger) {
	Logger = l
}
