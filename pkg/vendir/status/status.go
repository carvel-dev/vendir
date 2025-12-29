// Copyright 2025 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

type CompleteReference struct {
	SHA    string
	Tags   []string
	Others []string
}

type Status struct {
	DirectoryPath     string
	ContentPath       string
	TargetRef         string
	Ref               CompleteReference
	UncommitedChanges []string
	LocalCsets        []string
}

func (s Status) Path() string {
	return path.Join(s.DirectoryPath, s.ContentPath)
}

func (s Status) IsSafe() bool {
	return len(s.UncommitedChanges) == 0 && len(s.LocalCsets) == 0
}

func (s Status) MatchTarget() bool {
	return strings.HasPrefix(s.Ref.SHA, s.TargetRef) ||
		slices.Contains(s.Ref.Tags, s.TargetRef) ||
		slices.Contains(s.Ref.Others, s.TargetRef)
}

func (s Status) MatchTargetTag() bool {
	return slices.Contains(s.Ref.Tags, s.TargetRef)
}

func (s Status) String() string {
	messages := make([]string, 0, 3)

	if s.IsSafe() {
		messages = append(messages, "clean")
	}

	if len(s.UncommitedChanges) != 0 {
		messages = append(messages, fmt.Sprintf("%d uncommited changes", len(s.UncommitedChanges)))
	}
	if len(s.LocalCsets) != 0 {
		messages = append(messages, fmt.Sprintf("%d unpushed commits", len(s.LocalCsets)))
	}

	if !s.MatchTarget() {
		messages = append(messages, "ref mismatch")
	}

	return strings.Join(messages, ", ")
}

type StatusList []*Status

func (sm StatusList) String() string {
	var s string

	s += "Detailled status:\n"
	for _, status := range sm {
		s += "- " + status.DirectoryPath + "/" + status.ContentPath + ": " + status.String() + "\n"
	}

	if !sm.IsSafe() {
		s += "\n  /!\\ At least one directory is not clean /!\\\n"
	}

	return s
}

func (sm StatusList) IsSafe() bool {
	isSafe := true
	for _, status := range sm {
		if !status.IsSafe() {
			isSafe = false
			break
		}
	}

	return isSafe
}
