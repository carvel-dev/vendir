// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bytes"
	"fmt"

	goui "github.com/cppforlife/go-cli-ui/ui"
)

// Logger interface that defines the logger functions
type Logger interface {
	Logf(string, ...interface{})
}

// NewLogger creates a logger that will write to the ui when tty is activated
func NewLogger(ui goui.UI) *UILogger {
	return &UILogger{
		ui: ui,
	}
}

// NewLoggerNoTTY creates a logger that will write even when tty is deactivated
func NewLoggerNoTTY(ui goui.UI) *NoTTYLogger {
	return &NoTTYLogger{ui: ui}
}

// UILogger struct that interacts with the UI.
// This logger only writes to the UI when the tty is activated
type UILogger struct {
	ui goui.UI
}

// Logf Prints log to UI when tty is activated
func (n *UILogger) Logf(msg string, args ...interface{}) {
	n.ui.BeginLinef(msg, args...)
}

// NoTTYLogger struct that interacts with the UI.
// This logger allow writing to the UI when the tty is deactivated
type NoTTYLogger struct {
	ui goui.UI
}

// Logf Prints log to UI when tty is deactivated
func (n *NoTTYLogger) Logf(msg string, args ...interface{}) {
	n.ui.PrintBlock([]byte(fmt.Sprintf(msg, args...)))
}

// NewNoopLogger creates a new noop logger
func NewNoopLogger() *NoopLogger {
	return &NoopLogger{}
}

// NoopLogger this logger will not print
type NoopLogger struct{}

// Logf does nothing
func (n NoopLogger) Logf(string, ...interface{}) {}

// BufferLogger write logs to a buffer
type BufferLogger struct {
	buf *bytes.Buffer
}

// NewBufferLogger creates a new BufferLogger
func NewBufferLogger(buf *bytes.Buffer) *BufferLogger {
	return &BufferLogger{buf: buf}
}

// Logf writes log to the buffer
func (b *BufferLogger) Logf(msg string, args ...interface{}) {
	b.buf.Write([]byte(fmt.Sprintf(msg, args...)))
}
