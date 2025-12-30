// Copyright 2025 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package hg

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"

	ctlstatus "carvel.dev/vendir/pkg/vendir/status"
)

const emptyString = ""

func (d Sync) Status(target string) (*ctlstatus.Status, error) {
	_, err := os.Stat(path.Join(target, ".hg"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	hg, err := NewHg(d.opts, d.log, d.refFetcher, nil)
	if err != nil {
		return nil, fmt.Errorf("Setting up hg: %w", err)
	}
	defer hg.Close()

	out, _, err := hg.run([]string{
		"log", "-r", ".",
		"-T", "{node}\n{tags}\n{branch}\n{topic}\n{bookmarks}\n",
	}, target)
	if err != nil {
		return nil, err
	}

	splitted := strings.Split(out, "\n")
	sha := splitted[0]       //nolint:revive
	tags := splitted[1]      //nolint:revive
	branch := splitted[2]    //nolint:revive
	topic := splitted[3]     //nolint:revive
	bookmarks := splitted[4] //nolint:revive

	status := ctlstatus.Status{
		TargetRef: d.opts.Ref,
		Ref: ctlstatus.CompleteReference{
			SHA: sha,
		},
	}
	if tags != emptyString {
		status.Ref.Tags = strings.Split(tags, " ")
		status.Ref.Tags = slices.DeleteFunc(
			status.Ref.Tags, func(t string) bool { return t == "tip" })
	}
	if branch != emptyString {
		status.Ref.Others = append(status.Ref.Others, branch)
	}
	if topic != emptyString {
		status.Ref.Others = append(status.Ref.Others, topic)
	}
	if bookmarks != emptyString {
		status.Ref.Others = append(status.Ref.Others, bookmarks)
	}

	out, _, err = hg.run([]string{"status"}, target)
	if err != nil {
		return nil, err
	}
	if out != emptyString {
		status.UncommitedChanges = strings.Split(strings.TrimSpace(out), "\n")
	}

	out, _, err = hg.run([]string{"out", "-q", "-T", "{node}\n"}, target)
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
			return nil, err
		}
	}

	if out != emptyString {
		status.LocalCsets = strings.Split(strings.TrimSpace(out), "\n")
	}

	return &status, nil
}
