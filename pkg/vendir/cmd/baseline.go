// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	ctldir "carvel.dev/vendir/pkg/vendir/directory"
	ctlcache "carvel.dev/vendir/pkg/vendir/fetch/cache"
)

func NewBaselineOptions(ui ui.UI) *BaselineOptions { //nolint:revive
	return &BaselineOptions{ui: ui}
}

func NewBaselineCmd(o *BaselineOptions) *cobra.Command {
	cmd := cobra.Command{
		Use:   "baseline",
		Short: "Update git/hg repositories 'ref' to match the current commits",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
	}

	cmd.Flags().BoolVarP(
		&o.Yes, "yes", "y", false,
		"If true, automatically answer 'yes' to all the questions")

	cmd.Flags().StringSliceVarP(
		&o.Files, "file", "f", []string{defaultConfigName},
		"Set configuration file")
	cmd.Flags().StringVar(
		&o.LockFile, "lock-file", defaultLockName, "Set lock file")
	cmd.Flags().StringVar(
		&o.Chdir, "chdir", "", "Set current directory for process")
	cmd.Flags().BoolVar(
		&o.PreferSHA, "prefer-sha", false, "Prefer sha instead of tags")
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "List what would be done")

	return &cmd
}

type BaselineOptions struct {
	ui ui.UI

	Files    []string
	LockFile string

	Chdir    string
	ExitCode bool

	PreferSHA bool
	Yes       bool
	DryRun    bool
}

func (o *BaselineOptions) Run() error {
	if len(o.Chdir) > 0 { //nolint:revive
		err := os.Chdir(o.Chdir)
		if err != nil {
			return fmt.Errorf("Running chdir: %s", err)
		}
	}

	conf, secrets, configMaps, err := ctlconf.NewConfigFromFiles(o.Files)
	if err != nil {
		return (*SyncOptions)(nil).configReadHintErrMsg(err, o.Files)
	}

	existingLockConfig, err := ctlconf.NewLockConfigFromFile(o.LockFile)
	if err != nil {
		return err
	}

	cache, err := ctlcache.NewCache(os.Getenv("VENDIR_CACHE_DIR"), "0Mi")
	if err != nil {
		return fmt.Errorf("Unable to create cache: %s", err)
	}
	syncOpts := ctldir.SyncOpts{
		RefFetcher:     ctldir.NewNamedRefFetcher(secrets, configMaps),
		GithubAPIToken: os.Getenv("VENDIR_GITHUB_API_TOKEN"),
		HelmBinary:     os.Getenv("VENDIR_HELM_BINARY"),
		Cache:          cache,
		Lazy:           false,
		Partial:        false,
	}

	statusMap, err := fullStatus(conf, syncOpts, existingLockConfig, o.ui)
	if err != nil {
		return err
	}

	newRefs := make(map[string]string)

	for _, status := range statusMap {
		if !status.MatchTarget() ||
			!o.PreferSHA &&
				!status.MatchTargetTag() &&
				len(status.Ref.Tags) != 0 { //nolint:revive
			var newRef string
			if o.PreferSHA || len(status.Ref.Tags) == 0 { //nolint:revive
				newRef = status.Ref.SHA
			} else {
				newRef = status.Ref.Tags[0] //nolint:revive
			}
			newRefs[status.Path()] = newRef
		}
	}

	if len(newRefs) == 0 { //nolint:revive
		o.ui.PrintLinef(
			"All references already match current state, no update needed")

		return nil
	}

	o.ui.PrintLinef("New baseline:")
	for _, status := range statusMap {
		newRef := newRefs[status.Path()]
		if newRef != "" {
			newRef = " -> " + newRef
		}
		o.ui.PrintLinef(
			"%s/%s: %s%s",
			status.DirectoryPath, status.ContentPath, status.TargetRef, newRef)
	}

	if !o.DryRun {
		for _, fname := range o.Files {
			if err := updateRefs(fname, newRefs); err != nil {
				return err
			}
		}
	}

	return nil
}

func loadFile(fname string) (*yaml.Node, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	var node yaml.Node

	if err := yaml.NewDecoder(f).Decode(&node); err != nil {
		return nil, err
	}

	return &node, err
}

func saveFile(fname string, doc *yaml.Node) error {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2) //nolint:revive
	if err := enc.Encode(doc); err != nil {
		_ = f.Close()

		return err
	}

	return f.Close()
}

func updateRefs(fname string, newRefs map[string]string) error {
	doc, err := loadFile(fname)
	if err != nil {
		return err
	}

	if doc.Kind != yaml.DocumentNode {
		panic("expects the root node")
	}

	top := doc.Content[0] //nolint:revive
	if top.Kind != yaml.MappingNode {
		panic("top content must be a mapping")
	}

	directories := getMappingNodeChild(top, "directories")

	for _, d := range directories.Content {
		dirPath := getMappingNodeChild(d, "path").Value

		contents := getMappingNodeChild(d, "contents")

		for _, content := range contents.Content {
			contentPath := getMappingNodeChild(content, "path").Value

			fullPath := path.Join(dirPath, contentPath)

			if newRef, ok := newRefs[fullPath]; ok {
				if hg := getMappingNodeChild(content, "hg"); hg != nil {
					ref := getMappingNodeChild(hg, "ref")
					if ref == nil {
						return fmt.Errorf(
							"could not find 'ref' for '%s'", fullPath)
					}
					ref.Value = newRef
				}
				if git := getMappingNodeChild(content, "git"); git != nil {
					ref := getMappingNodeChild(git, "ref")
					if ref == nil {
						return fmt.Errorf(
							"could not find 'ref' for '%s'", fullPath)
					}
					ref.Value = newRef
				}
			}
		}
	}

	return saveFile(fname, doc)
}

func getMappingNodeChild(node *yaml.Node, name string) *yaml.Node {
	for i := 0; i < len(node.Content); i += 2 { //nolint:revive
		if node.Content[i].Value == name {
			return node.Content[i+1] //nolint:revive
		}
	}

	return nil
}
