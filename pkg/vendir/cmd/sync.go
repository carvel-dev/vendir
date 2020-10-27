// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	ctlconf "github.com/k14s/vendir/pkg/vendir/config"
	ctldir "github.com/k14s/vendir/pkg/vendir/directory"
	"github.com/spf13/cobra"
)

const (
	defaultConfigName = "vendir.yml"
	defaultLockName   = "vendir.lock.yml"
)

type SyncOptions struct {
	ui ui.UI

	File        string
	Directories []string
	Locked      bool
}

func NewSyncOptions(ui ui.UI) *SyncOptions {
	return &SyncOptions{ui: ui}
}

func NewSyncCmd(o *SyncOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync directories",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
	}
	cmd.Flags().StringVarP(&o.File, "file", "f", defaultConfigName, "Set configuration file")
	cmd.Flags().StringSliceVarP(&o.Directories, "directory", "d", nil, "Sync specific directory (format: dir/sub-dir[=local-dir])")
	cmd.Flags().BoolVarP(&o.Locked, "locked", "l", false, "Consult lock file to pull exact references (e.g. use git sha instead of branch name)")
	return cmd
}

func (o *SyncOptions) Run() error {
	conf, err := ctlconf.NewConfigFromFile(o.File)
	if err != nil {
		return err
	}

	dirs, err := o.directories()
	if err != nil {
		return err
	}

	err = o.applyUseDirectories(&conf, dirs)
	if err != nil {
		return err
	}

	if len(dirs) > 0 {
		conf, err = conf.Subset(dirOverrides(dirs).Paths())
		if err != nil {
			return err
		}

		configBs, err := conf.AsBytes()
		if err != nil {
			return err
		}

		o.ui.PrintLinef("Config with overrides")
		o.ui.PrintBlock(configBs)
	}

	if o.Locked {
		lockConfig, err := ctlconf.NewLockConfigFromFile(defaultLockName)
		if err != nil {
			return err
		}

		err = conf.Lock(lockConfig)
		if err != nil {
			return err
		}

		configBs, err := conf.AsBytes()
		if err != nil {
			return err
		}

		o.ui.PrintLinef("Config with locks")
		o.ui.PrintBlock(configBs)
	}

	shouldWriteLockConfig := (o.File == defaultConfigName) && len(dirs) == 0
	lockConfig := ctlconf.NewLockConfig()

	syncOpts := ctldir.SyncOpts{GithubAPIToken: os.Getenv("VENDIR_GITHUB_API_TOKEN")}

	for _, dirConf := range conf.Directories {
		dirLockConf, err := ctldir.NewDirectory(dirConf, o.ui).Sync(syncOpts)
		if err != nil {
			return fmt.Errorf("Syncing directory '%s': %s", dirConf.Path, err)
		}

		lockConfig.Directories = append(lockConfig.Directories, dirLockConf)
	}

	lockConfigBs, err := lockConfig.AsBytes()
	if err != nil {
		return err
	}

	o.ui.PrintLinef("Lock config")
	o.ui.PrintBlock(lockConfigBs)

	if !shouldWriteLockConfig {
		o.ui.PrintLinef("Lock config is not saved to '%s' due to command line overrides", defaultLockName)
		return nil
	}

	return lockConfig.WriteToFile(defaultLockName)
}

func (o *SyncOptions) directories() ([]dirOverride, error) {
	var dirs []dirOverride

	for _, val := range o.Directories {
		pieces := strings.SplitN(val, "=", 2)
		if len(pieces) == 1 {
			dirs = append(dirs, dirOverride{Path: pieces[0]})
		} else {
			dirs = append(dirs, dirOverride{Path: pieces[0], LocalDir: pieces[1]})
		}
	}

	dirOverrides(dirs).ExpandUserHomeDirs()

	return dirs, nil
}

func (o *SyncOptions) applyUseDirectories(conf *ctlconf.Config, dirs []dirOverride) error {
	for _, dir := range dirs {
		if len(dir.LocalDir) == 0 {
			continue
		}
		err := conf.UseDirectory(dir.Path, dir.LocalDir)
		if err != nil {
			return fmt.Errorf("Overriding '%s' with local directory: %s", dir.Path, err)
		}
	}
	return nil
}

type dirOverride struct {
	Path     string
	LocalDir string
}

type dirOverrides []dirOverride

func (dirs dirOverrides) Paths() []string {
	var result []string
	for _, d := range dirs {
		result = append(result, d.Path)
	}
	return result
}

func (dirs dirOverrides) ExpandUserHomeDirs() error {
	homeDir, expandErr := dirs.userHomeDir()

	for i, dir := range dirs {
		if len(dir.LocalDir) > 0 {
			// TODO does not support ~user convention
			if strings.HasPrefix(dir.LocalDir, "~") {
				if len(homeDir) == 0 && expandErr != nil {
					return expandErr
				}
				dir.LocalDir = filepath.Join(homeDir, dir.LocalDir[1:])
				dirs[i] = dir
			}
		}
	}

	return nil
}

func (dirOverrides) userHomeDir() (string, error) {
	out, err := exec.Command("sh", "-c", "echo ~").Output()
	if err != nil {
		return "", fmt.Errorf("Expanding user home directory: %s", err)
	}
	return strings.TrimSpace(string(out)), nil
}
