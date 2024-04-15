// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1

// ErrIsBundle Error when the provided Image Reference is a Bundle
type ErrIsBundle struct{}

// Error message
func (e ErrIsBundle) Error() string {
	return "The provided image is a bundle"
}

// Is check if the error is of the type ErrIsBundle
func (e ErrIsBundle) Is(target error) bool {
	_, ok := target.(*ErrIsBundle)
	return ok
}

// ErrIsNotBundle Error when the provided Image Reference is not a Bundle
type ErrIsNotBundle struct{}

// Error message
func (e ErrIsNotBundle) Error() string {
	return "The provided image is not a bundle"
}

// Is check if the error is of the type ErrIsNotBundle
func (e ErrIsNotBundle) Is(target error) bool {
	_, ok := target.(*ErrIsNotBundle)
	return ok
}
