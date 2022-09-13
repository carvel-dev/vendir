// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

// LogLevel specifies logging level (i.e. DEBUG, WARN)
type LogLevel int

// LoggerWithLevels wraps a ui.UI with logging levels
type LoggerWithLevels interface {
	Errorf(msg string, args ...interface{})
	Warnf(msg string, args ...interface{})
	Debugf(msg string, args ...interface{})
	Tracef(msg string, args ...interface{})
	Logf(msg string, args ...interface{})
}

// LogLevelRetriever retrieves the log level
type LogLevelRetriever interface {
	Level() LogLevel
}

const (
	// LogTrace most verbose level
	LogTrace LogLevel = iota
	// LogDebug used when more information than normal is needed
	LogDebug LogLevel = iota
	// LogWarn only logs warnings and errors
	LogWarn LogLevel = iota
)

// NewIndentedLevelLogger creates a new logger with levels and indented by 2 spaces
func NewIndentedLevelLogger(logger LoggerWithLevels) *LevelLogger {
	level := LogWarn
	if l, ok := logger.(LogLevelRetriever); ok {
		level = l.Level()
	}

	return &LevelLogger{
		logger:   NewPrefixedLogger("  ", logger),
		LogLevel: level,
	}
}

// NewUILevelLogger is a LevelLogger constructor, wrapping a ui.UI with a specific log level
func NewUILevelLogger(level LogLevel, logger Logger) *LevelLogger {
	return &LevelLogger{
		logger:   logger,
		LogLevel: level,
	}
}

// NewNoopLevelLogger will not print anything
func NewNoopLevelLogger() *LevelLogger {
	return &LevelLogger{
		logger:   NewNoopLogger(),
		LogLevel: LogWarn,
	}
}

// LevelLogger allows specifying a log level to a ui.UI
type LevelLogger struct {
	logger   Logger
	LogLevel LogLevel
}

// Errorf used to log error related messages
func (l LevelLogger) Errorf(msg string, args ...interface{}) {
	l.Logf("Error: "+msg, args...)
}

// Warnf used to log warning related messages
func (l LevelLogger) Warnf(msg string, args ...interface{}) {
	if l.LogLevel <= LogWarn {
		l.Logf("Warning: "+msg, args...)
	}
}

// Logf logs the provided message
func (l LevelLogger) Logf(msg string, args ...interface{}) {
	l.logger.Logf(msg, args...)
}

// Debugf used to log debug related messages
func (l LevelLogger) Debugf(msg string, args ...interface{}) {
	if l.LogLevel <= LogDebug {
		l.Logf(msg, args...)
	}
}

// Tracef used to log trace related messages
func (l LevelLogger) Tracef(msg string, args ...interface{}) {
	if l.LogLevel == LogTrace {
		l.Logf(msg, args...)
	}
}

// Level retrieve the current log level for this logger
func (l LevelLogger) Level() LogLevel {
	return l.LogLevel
}
