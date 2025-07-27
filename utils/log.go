package utils

import (
	"OJ-API/config"
	"fmt"
	"log"
	"os"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var currentLogLevel LogLevel

// Logger wrapper
type Logger struct {
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
}

var logger *Logger

func GetCurrentLogLevel() LogLevel {
	return currentLogLevel
}

// InitLog initializes the logger with the specified log level.
func InitLog() {
	logLevel := config.Config("LOG_LEVEL")

	// Create loggers with different prefixes
	logger = &Logger{
		debugLogger: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
		infoLogger:  log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		warnLogger:  log.New(os.Stdout, "[WARN] ", log.LstdFlags),
		errorLogger: log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile),
	}

	switch logLevel {
	case "debug":
		currentLogLevel = DEBUG
		Info("Log level set to DEBUG")
	case "info":
		currentLogLevel = INFO
		Info("Log level set to INFO")
	case "warn":
		currentLogLevel = WARN
		Info("Log level set to WARN")
	case "error":
		currentLogLevel = ERROR
		Info("Log level set to ERROR")
	default:
		logger.errorLogger.Output(2, fmt.Sprintf("Unknown log level: %s", logLevel))
		os.Exit(1)
	}
}

// Debug logs debug messages
func Debug(v ...interface{}) {
	if currentLogLevel <= DEBUG {
		logger.debugLogger.Output(2, fmt.Sprint(v...))
	}
}

// Debugf logs debug messages with formatting
func Debugf(format string, v ...interface{}) {
	if currentLogLevel <= DEBUG {
		logger.debugLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Info logs info messages
func Info(v ...interface{}) {
	if currentLogLevel <= INFO {
		logger.infoLogger.Output(2, fmt.Sprint(v...))
	}
}

// Infof logs info messages with formatting
func Infof(format string, v ...interface{}) {
	if currentLogLevel <= INFO {
		logger.infoLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Warn logs warning messages
func Warn(v ...interface{}) {
	if currentLogLevel <= WARN {
		logger.warnLogger.Output(2, fmt.Sprint(v...))
	}
}

// Warnf logs warning messages with formatting
func Warnf(format string, v ...interface{}) {
	if currentLogLevel <= WARN {
		logger.warnLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Error logs error messages
func Error(v ...interface{}) {
	if currentLogLevel <= ERROR {
		logger.errorLogger.Output(2, fmt.Sprint(v...))
	}
}

// Errorf logs error messages with formatting
func Errorf(format string, v ...interface{}) {
	if currentLogLevel <= ERROR {
		logger.errorLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Fatal logs fatal messages and exits
func Fatal(v ...interface{}) {
	logger.errorLogger.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

// Fatalf logs fatal messages with formatting and exits
func Fatalf(format string, v ...interface{}) {
	logger.errorLogger.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}
