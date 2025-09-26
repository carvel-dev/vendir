// Copyright 2025 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"fmt"
	"strings"
)

type CompleteReference struct {
	SHA    string
	Tags   []string
	Others []string
}

type Status struct {
	Ref               CompleteReference
	UncommitedChanges []string
	LocalCsets        []string
}

func (s Status) IsSafe() bool {
	return len(s.UncommitedChanges) == 0 && len(s.LocalCsets) == 0
}

func (s Status) String() string {
	if s.IsSafe() {
		return "clean"
	}

	messages := make([]string, 0, 2)

	if len(s.UncommitedChanges) != 0 {
		messages = append(messages, fmt.Sprintf("%d uncommited changes", len(s.UncommitedChanges)))
	}
	if len(s.LocalCsets) != 0 {
		messages = append(messages, fmt.Sprintf("%d unpushed commits", len(s.LocalCsets)))
	}

	return strings.Join(messages, ", ")
}

type StatusMap map[string]*Status

func (sm StatusMap) String() string {
	var s string

	s += "Detailled status:\n"
	for dir, status := range sm {
		s += "- " + dir + ": " + status.String() + "\n"
	}

	if !sm.IsSafe() {
		s += "\n  /!\\ At least one directory is not clean /!\\\n"
	}

	return s
}

func (sm StatusMap) IsSafe() bool {
	isSafe := true
	for _, status := range sm {
		if !status.IsSafe() {
			isSafe = false
			break
		}
	}

	return isSafe
}
