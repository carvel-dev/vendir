// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"time"

	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

type NonRetryableError struct {
	Message string
}

func (n NonRetryableError) Error() string {
	return n.Message
}

func Retry(doFunc func() error) error {
	var lastErr error

	for i := 0; i < 5; i++ {
		lastErr = doFunc()
		if lastErr == nil {
			return nil
		}

		if tranErr, ok := lastErr.(*transport.Error); ok {
			if len(tranErr.Errors) > 0 {
				if tranErr.Errors[0].Code == transport.UnauthorizedErrorCode {
					return fmt.Errorf("Non-retryable error: %s", lastErr)
				}
			}
		}
		if nonRetryableError, ok := lastErr.(NonRetryableError); ok {
			return nonRetryableError
		}

		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("Retried 5 times: %s", lastErr)
}
