// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"fmt"
	"os"

	pb "github.com/cheggaaa/pb/v3"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/mattn/go-isatty"
)

// ProgressLogger Progress bar
type ProgressLogger interface {
	Start(ctx context.Context, progress <-chan regv1.Update)
	End()
}

// NewProgressBar constructor to build a ProgressLogger responsible for printing out a progress bar using updates when
// writing to a registry via ggcr
func NewProgressBar(logger LoggerWithLevels, finalMessage, errorMessagePrefix string) ProgressLogger {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		return &ProgressBarLogger{logger: logger, finalMessage: finalMessage, errorMessagePrefix: errorMessagePrefix}
	}

	return &ProgressBarNoTTYLogger{logger: logger, finalMessage: finalMessage}
}

// NewNoopProgressBar constructs a Noop Progress bar that will not display anything
func NewNoopProgressBar() ProgressLogger {
	return &ProgressBarNoTTYLogger{}
}

// ProgressBarLogger display progress bar on output
type ProgressBarLogger struct {
	cancelFunc         context.CancelFunc
	bar                *pb.ProgressBar
	logger             LoggerWithLevels
	finalMessage       string
	errorMessagePrefix string
}

// Start the display of the Progress Bar
func (l *ProgressBarLogger) Start(ctx context.Context, progressChan <-chan regv1.Update) {
	ctx, cancelFunc := context.WithCancel(ctx)
	l.cancelFunc = cancelFunc
	// Add a new empty line to separate the progress bar from prior output
	fmt.Println()
	l.bar = pb.New64(0)
	l.bar.Set(pb.Bytes, true)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-progressChan:
				if update.Error != nil {
					l.logger.Errorf("%s: %s\n", l.errorMessagePrefix, update.Error)
					continue
				}

				if update.Total == 0 {
					return
				}
				if !l.bar.IsStarted() {
					l.bar.SetTotal(update.Total)
					l.bar.Start()
				}
				l.bar.SetCurrent(update.Complete)
				l.bar.Write()
			}
		}
	}()
}

// End stops the progress bar and writes the final message
func (l *ProgressBarLogger) End() {
	if l.cancelFunc != nil {
		l.cancelFunc()
	}
	l.bar.Finish()
	l.logger.Logf("\n%s", l.finalMessage)
}

// ProgressBarNoTTYLogger does not display the progress bar
type ProgressBarNoTTYLogger struct {
	cancelFunc   context.CancelFunc
	logger       Logger
	finalMessage string
}

// Start consuming the progress channel but does not display anything
func (l *ProgressBarNoTTYLogger) Start(ctx context.Context, progressChan <-chan regv1.Update) {
	ctx, cancelFunc := context.WithCancel(ctx)
	l.cancelFunc = cancelFunc
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-progressChan:
			}
		}
	}()
}

// End Write the final message
func (l *ProgressBarNoTTYLogger) End() {
	if l.cancelFunc != nil {
		l.cancelFunc()
	}
	if l.logger != nil && l.finalMessage != "" {
		l.logger.Logf(l.finalMessage)
	}
}
