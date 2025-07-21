package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[90m"
	
	// Bright colors
	ColorBrightRed    = "\033[91m"
	ColorBrightGreen  = "\033[92m"
	ColorBrightYellow = "\033[93m"
	ColorBrightBlue   = "\033[94m"
	ColorBrightPurple = "\033[95m"
	ColorBrightCyan   = "\033[96m"
	ColorBrightWhite  = "\033[97m"
)

// LogLevel represents different log levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ColoredLogger provides colored logging with different contexts
type ColoredLogger struct {
	context    string
	color      string
	level      LogLevel
	showCaller bool
}

// NewColoredLogger creates a new colored logger with context
func NewColoredLogger(context, color string) *ColoredLogger {
	return &ColoredLogger{
		context:    context,
		color:      color,
		level:      INFO,
		showCaller: false,
	}
}

// SetLevel sets the minimum log level
func (l *ColoredLogger) SetLevel(level LogLevel) {
	l.level = level
}

// SetShowCaller enables/disables showing caller information
func (l *ColoredLogger) SetShowCaller(show bool) {
	l.showCaller = show
}

// formatMessage formats a log message with color and context
func (l *ColoredLogger) formatMessage(level LogLevel, format string, args ...interface{}) string {
	timestamp := time.Now().Format("15:04:05.000")
	
	// Get caller info if enabled
	caller := ""
	if l.showCaller {
		if pc, file, line, ok := runtime.Caller(3); ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				parts := strings.Split(file, "/")
				fileName := parts[len(parts)-1]
				caller = fmt.Sprintf(" %s:%d", fileName, line)
			}
		}
	}
	
	// Format the message
	message := fmt.Sprintf(format, args...)
	
	// Choose level color
	levelColor := ""
	switch level {
	case DEBUG:
		levelColor = ColorGray
	case INFO:
		levelColor = ColorBlue
	case WARN:
		levelColor = ColorYellow
	case ERROR:
		levelColor = ColorRed
	case FATAL:
		levelColor = ColorBrightRed
	}
	
	// Format: [timestamp] [context] [level] message [caller]
	return fmt.Sprintf("%s[%s]%s %s[%s]%s %s[%s]%s %s%s%s",
		ColorGray, timestamp, ColorReset,
		l.color, l.context, ColorReset,
		levelColor, level.String(), ColorReset,
		message,
		ColorGray, caller,
	)
}

// log is the internal logging function
func (l *ColoredLogger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	
	message := l.formatMessage(level, format, args...)
	
	if level == FATAL {
		log.Println(message)
		os.Exit(1)
	} else {
		log.Println(message)
	}
}

// Debug logs a debug message
func (l *ColoredLogger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs an info message
func (l *ColoredLogger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs a warning message
func (l *ColoredLogger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs an error message
func (l *ColoredLogger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Fatal logs a fatal message and exits
func (l *ColoredLogger) Fatal(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
}

// Printf implements the Printf interface for compatibility
func (l *ColoredLogger) Printf(format string, args ...interface{}) {
	l.Info(format, args...)
}

// Println implements the Println interface for compatibility
func (l *ColoredLogger) Println(args ...interface{}) {
	l.Info(fmt.Sprint(args...))
}

// Predefined loggers for different components
var (
	ServerLogger = NewColoredLogger("SERVER", ColorBrightGreen)
	ClientLogger = NewColoredLogger("CLIENT", ColorBrightBlue)
	GameLogger   = NewColoredLogger("GAME", ColorBrightPurple)
	AILogger     = NewColoredLogger("AI", ColorBrightCyan)
	TestLogger   = NewColoredLogger("TEST", ColorBrightYellow)
)

// InitLoggers initializes all loggers with appropriate settings
func InitLoggers(level LogLevel, showCaller bool) {
	loggers := []*ColoredLogger{
		ServerLogger,
		ClientLogger,
		GameLogger,
		AILogger,
		TestLogger,
	}
	
	for _, logger := range loggers {
		logger.SetLevel(level)
		logger.SetShowCaller(showCaller)
	}
}

// CreatePlayerLogger creates a colored logger for a specific player
func CreatePlayerLogger(playerName, color string) *ColoredLogger {
	return NewColoredLogger(fmt.Sprintf("PLAYER:%s", playerName), color)
}

// CreateAILogger creates a colored logger for a specific AI strategy
func CreateAILogger(strategy, color string) *ColoredLogger {
	return NewColoredLogger(fmt.Sprintf("AI:%s", strategy), color)
}