package git

import (
	"os"
	"path"
	"strings"

	ctlstatus "carvel.dev/vendir/pkg/vendir/status"
)

func (d Sync) Status(target string) (*ctlstatus.Status, error) {
	_, err := os.Stat(path.Join(target, ".git"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	git := NewGit(d.opts, d.log, d.refFetcher)

	status := ctlstatus.Status{
		TargetRef: d.opts.Ref,
	}

	out, _, err := git.cmdRunner.Run([]string{"rev-parse", "HEAD"}, []string{}, target)
	if err != nil {
		return nil, err
	}
	status.Ref.SHA = strings.TrimSpace(out)

	out, _, err = git.cmdRunner.Run([]string{"tag", "--contains"}, []string{}, target)
	if err != nil {
		return nil, err
	}
	if out != "" {
		status.Ref.Tags = strings.Split(strings.TrimSpace(out), "\n")
	}

	out, _, err = git.cmdRunner.Run([]string{"branch", "--contains"}, []string{}, target)
	if err != nil {
		return nil, err
	}
	if out != "" {
		status.Ref.Others = strings.Split(out, "\n")
	}

	out, _, err = git.cmdRunner.Run(
		[]string{"log", "--branches", "--not", "--remotes", "--oneline"},
		[]string{},
		target,
	)
	if err != nil {
		return nil, err
	}

	if out != "" {
		status.LocalCsets = strings.Split(strings.TrimSpace(out), "\n")
	}

	out, _, err = git.cmdRunner.Run(
		[]string{"status", "--short"},
		[]string{},
		target,
	)
	if err != nil {
		return nil, err
	}

	if out != "" {
		status.UncommitedChanges = strings.Split(strings.TrimSpace(out), "\n")
	}

	return &status, nil
}
