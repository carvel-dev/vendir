package ui_test

import (
	"bytes"
	"fmt"

	. "github.com/cppforlife/go-cli-ui/ui"
)

type RecordingLogger struct {
	ErrOut   *bytes.Buffer
	DebugOut *bytes.Buffer
}

var _ ExternalLogger = &RecordingLogger{}

func NewRecordingLogger() *RecordingLogger {
	return &RecordingLogger{ErrOut: bytes.NewBufferString(""), DebugOut: bytes.NewBufferString("")}
}

func (l *RecordingLogger) Error(tag, msg string, args ...interface{}) {
	fmt.Fprintf(l.ErrOut, fmt.Sprintf(fmt.Sprintf("%s: %s\n", tag, msg), args...))
}

func (l *RecordingLogger) Debug(tag, msg string, args ...interface{}) {
	fmt.Fprintf(l.DebugOut, fmt.Sprintf(fmt.Sprintf("%s: %s\n", tag, msg), args...))
}
