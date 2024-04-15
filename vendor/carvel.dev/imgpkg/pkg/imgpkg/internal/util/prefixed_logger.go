// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bytes"
	"fmt"
	"sync"

	goui "github.com/cppforlife/go-cli-ui/ui"
)

// NewIndentedLogger creates a new logger indented by 2 spaces
func NewIndentedLogger(logger Logger) *PrefixedLogger {
	return NewPrefixedLogger("  ", logger)
}

// NewPrefixedLogger constructor for building a UI with a prefix when logging a message
func NewPrefixedLogger(prefix string, logger Logger) *PrefixedLogger {
	return &PrefixedLogger{logger, prefix, &sync.Mutex{}}
}

// PrefixedLogger Logger that attached a prefix to each line
type PrefixedLogger struct {
	parent     Logger
	prefix     string
	writerLock *sync.Mutex
}

// Logf logs message provided
// adds the prefix to each new line of the msg parameter
func (p PrefixedLogger) Logf(msg string, args ...interface{}) {
	data := fmt.Sprintf(msg, args...)
	newData := make([]byte, len(data))
	copy(newData, data)

	endsWithNl := bytes.HasSuffix(newData, []byte("\n"))
	if endsWithNl {
		newData = newData[0 : len(newData)-1]
	}
	newData = bytes.Replace(newData, []byte("\n"), []byte("\n"+p.prefix), -1)
	newData = append(newData, []byte("\n")...)
	newData = append([]byte(p.prefix), newData...)

	p.writerLock.Lock()
	defer p.writerLock.Unlock()

	p.parent.Logf(string(newData))
}

// UIPrefixWriter prints a prefix when the underlying ui prints a message
type UIPrefixWriter struct {
	goui.UI
	prefix     string
	writerLock *sync.Mutex
}

// BeginLinef writes a message and args adding a configured prefix
func (w *UIPrefixWriter) BeginLinef(msg string, args ...interface{}) {
	_, err := w.Write([]byte(fmt.Sprintf(msg, args...)))
	if err != nil {
		panic(fmt.Sprintf("Unable to write to ui: %s", err))
	}
}

func (w *UIPrefixWriter) Write(data []byte) (int, error) {
	newData := make([]byte, len(data))
	copy(newData, data)

	endsWithNl := bytes.HasSuffix(newData, []byte("\n"))
	if endsWithNl {
		newData = newData[0 : len(newData)-1]
	}
	newData = bytes.Replace(newData, []byte("\n"), []byte("\n"+w.prefix), -1)
	newData = append(newData, []byte("\n")...)
	newData = append([]byte(w.prefix), newData...)

	w.writerLock.Lock()
	defer w.writerLock.Unlock()

	w.PrintBlock(newData)

	// return original data length
	return len(data), nil
}
