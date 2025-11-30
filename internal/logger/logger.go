package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// Level represents a log level
type Level int

const (
	// LevelDebug is for detailed debug information
	LevelDebug Level = iota
	// LevelInfo is for general informational messages
	LevelInfo
	// LevelWarn is for warning messages
	LevelWarn
	// LevelError is for error messages
	LevelError
)

// String returns the string representation of a log level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger is a leveled logger
type Logger struct {
	level  Level
	logger *log.Logger
}

var defaultLogger *Logger

func init() {
	defaultLogger = New(os.Getenv("LOG_LEVEL"))
}

// New creates a new logger with the specified level
// levelStr can be: "debug", "info", "warn", "error" (case-insensitive)
// If empty or invalid, defaults to INFO level
func New(levelStr string) *Logger {
	level := parseLevel(levelStr)
	return &Logger{
		level:  level,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// NewWithWriter creates a new logger with a custom writer
func NewWithWriter(levelStr string, w io.Writer) *Logger {
	level := parseLevel(levelStr)
	return &Logger{
		level:  level,
		logger: log.New(w, "", log.LstdFlags),
	}
}

// parseLevel parses a log level string
func parseLevel(levelStr string) Level {
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug", "true", "1":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() Level {
	return l.level
}

// log outputs a log message at the specified level
func (l *Logger) log(level Level, format string, v ...interface{}) {
	if level < l.level {
		return
	}
	prefix := fmt.Sprintf("[%s] ", level)
	l.logger.Printf(prefix+format, v...)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(LevelDebug, format, v...)
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(LevelInfo, format, v...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	l.log(LevelWarn, format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(LevelError, format, v...)
}

// Default package-level functions using the default logger

// SetLevel sets the log level for the default logger
func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

// SetLevelFromString sets the log level from a string
func SetLevelFromString(levelStr string) {
	defaultLogger.level = parseLevel(levelStr)
}

// GetLevel returns the current log level
func GetLevel() Level {
	return defaultLogger.GetLevel()
}

// Debug logs a debug message using the default logger
func Debug(format string, v ...interface{}) {
	defaultLogger.Debug(format, v...)
}

// Info logs an info message using the default logger
func Info(format string, v ...interface{}) {
	defaultLogger.Info(format, v...)
}

// Warn logs a warning message using the default logger
func Warn(format string, v ...interface{}) {
	defaultLogger.Warn(format, v...)
}

// Error logs an error message using the default logger
func Error(format string, v ...interface{}) {
	defaultLogger.Error(format, v...)
}
