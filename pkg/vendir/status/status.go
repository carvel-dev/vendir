// Copyright 2025 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui/table"
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

func (s Status) toRow() []table.Value {
	row := make([]table.Value, 5)

	row[0] = table.NewValueString(s.DirectoryPath + "/" + s.ContentPath)

	if len(s.UncommitedChanges) != 0 {
		row[1] = table.NewValueInt(len(s.UncommitedChanges))
	}

	if len(s.LocalCsets) != 0 {
		row[2] = table.NewValueInt(len(s.LocalCsets))
	}

	if s.MatchTarget() {
		row[3] = table.NewValueFmt(table.NewValueString("up-to-date"), false)
	} else {
		row[3] = table.NewValueFmt(table.NewValueString("mismatch"), true)
	}

	if s.IsSafe() {
		row[4] = table.NewValueFmt(table.NewValueString("safe"), false)
	} else {
		row[4] = table.NewValueFmt(table.NewValueString("not safe"), true)
	}

	return row
}

type StatusList []*Status

func (sm StatusList) String() string {
	var s string

	s += "Detailed status:\n"
	for _, status := range sm {
		s += "- " + status.DirectoryPath + "/" + status.ContentPath + ": " + status.String() + "\n"
	}

	if !sm.IsSafe() {
		s += "\n  /!\\ At least one directory is not clean /!\\\n"
	}

	return s
}

func (sm StatusList) Table() table.Table {
	t := table.Table{
		Title: "Detailled status",
		Header: []table.Header{{
			Key:   "path",
			Title: "path",
		}, {
			Key:   "changes",
			Title: "changes",
		}, {
			Key:   "commits",
			Title: "commits",
		}, {
			Key:   "match",
			Title: "match",
		}, {
			Key:   "safe",
			Title: "safe",
		}},
	}

	for _, status := range sm {
		t.Rows = append(t.Rows, status.toRow())
	}

	return t
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
