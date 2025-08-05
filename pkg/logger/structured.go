package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// StructuredLogger provides JSON-formatted structured logging
type StructuredLogger struct {
	component string
	fields    map[string]interface{}
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(component string) *StructuredLogger {
	return &StructuredLogger{
		component: component,
		fields:    make(map[string]interface{}),
	}
}

// WithField adds a field to the logger
func (l *StructuredLogger) WithField(key string, value interface{}) *StructuredLogger {
	newLogger := &StructuredLogger{
		component: l.component,
		fields:    make(map[string]interface{}),
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value
	return newLogger
}

// WithFields adds multiple fields to the logger
func (l *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	newLogger := &StructuredLogger{
		component: l.component,
		fields:    make(map[string]interface{}),
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// log writes a log entry
func (l *StructuredLogger) log(level string, message string, args ...interface{}) {
	entry := make(map[string]interface{})
	entry["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	entry["level"] = level
	entry["component"] = l.component
	entry["message"] = fmt.Sprintf(message, args...)
	
	// Add custom fields
	for k, v := range l.fields {
		entry[k] = v
	}
	
	// Output as JSON
	data, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stdout, string(data))
}

// Info logs an info message
func (l *StructuredLogger) Info(message string, args ...interface{}) {
	l.log("info", message, args...)
}

// Warn logs a warning message
func (l *StructuredLogger) Warn(message string, args ...interface{}) {
	l.log("warn", message, args...)
}

// Error logs an error message
func (l *StructuredLogger) Error(message string, args ...interface{}) {
	l.log("error", message, args...)
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(message string, args ...interface{}) {
	l.log("debug", message, args...)
}

// Fatal logs a fatal message and exits
func (l *StructuredLogger) Fatal(message string, args ...interface{}) {
	l.log("fatal", message, args...)
	os.Exit(1)
}