package logger

import (
	"fmt"
)

// LoggerAdapter adapts ColoredLogger to various interfaces
type LoggerAdapter struct {
	*ColoredLogger
}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter(logger *ColoredLogger) *LoggerAdapter {
	return &LoggerAdapter{ColoredLogger: logger}
}

// Fatal implements handlers.Logger interface (without format string)
func (l *LoggerAdapter) Fatal(v ...interface{}) {
	l.ColoredLogger.Fatal(fmt.Sprint(v...))
}

// Fatalf implements the original interface
func (l *LoggerAdapter) Fatalf(format string, v ...interface{}) {
	l.ColoredLogger.Fatal(format, v...)
}

// Print implements standard print interface
func (l *LoggerAdapter) Print(v ...interface{}) {
	l.ColoredLogger.Info(fmt.Sprint(v...))
}

// AsHandlersLogger returns the adapter as a handlers.Logger interface
func AsHandlersLogger(logger *ColoredLogger) *LoggerAdapter {
	return NewLoggerAdapter(logger)
}