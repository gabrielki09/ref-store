package logger

import (
	"os"
	"time"

	"charm.land/log/v2"
)

var instance = log.NewWithOptions(os.Stderr, log.Options{
	ReportCaller:    true,
	ReportTimestamp: true,
	TimeFormat:      time.Kitchen,
})

func Error(msg any, elemt ...any) {
	instance.Helper()
	instance.Error(msg, elemt...)
}

func Info(msg string, data ...any) {
	instance.Helper()
	instance.Info(msg, data...)
}

func Fatal(msg string, data ...any) {
	instance.Helper()
	instance.Fatal(msg, data...)
}

func Debug(msg string, data ...any) {
	instance.Helper()
	instance.Debug(msg, data...)
}

func Warn(msg string, data ...any) {
	instance.Helper()
	instance.Warn(msg, data...)
}
