package hg

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	ctlstatus "carvel.dev/vendir/pkg/vendir/status"
)

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
	sha := splitted[0]
	tags := splitted[1]
	branch := splitted[2]
	topic := splitted[3]
	bookmarks := splitted[4]

	status := ctlstatus.Status{
		TargetRef: d.opts.Ref,
		Ref: ctlstatus.CompleteReference{
			SHA: sha,
		},
	}
	if tags != "" {
		status.Ref.Tags = strings.Split(tags, " ")
	}
	if branch != "" {
		status.Ref.Others = append(status.Ref.Others, branch)
	}
	if topic != "" {
		status.Ref.Others = append(status.Ref.Others, topic)
	}
	if bookmarks != "" {
		status.Ref.Others = append(status.Ref.Others, bookmarks)
	}

	out, _, err = hg.run([]string{"status"}, target)
	if err != nil {
		return nil, err
	}
	if out != "" {
		status.UncommitedChanges = strings.Split(strings.TrimSpace(out), "\n")
	}

	out, _, err = hg.run([]string{"out", "-q", "-T", "{node}\n"}, target)
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
			return nil, err
		}
	}

	if out != "" {
		status.LocalCsets = strings.Split(strings.TrimSpace(out), "\n")
	}

	return &status, nil
}
