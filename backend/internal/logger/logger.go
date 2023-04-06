package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var defaultLogger = &logrus.Logger{
	Out:       os.Stdout,
	Formatter: new(logrus.JSONFormatter),
	Level:     logrus.InfoLevel,
}

// Info logs message at Info level.
func Info(msg string) {
	defaultLogger.Infoln(msg)
}

// Error logs errors at Error level.
func Error(err error) {
	defaultLogger.Errorln(err)
}

// Fatal logs errorss at Fatal level.
func Fatal(err error) {
	defaultLogger.Fatalln(err)
}
