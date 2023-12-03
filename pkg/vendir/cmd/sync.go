// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	ctldir "carvel.dev/vendir/pkg/vendir/directory"
	ctlcache "carvel.dev/vendir/pkg/vendir/fetch/cache"
	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

const (
	defaultConfigName = "vendir.yml"
	defaultLockName   = "vendir.lock.yml"
)

type SyncOptions struct {
	ui ui.UI

	Files    []string
	LockFile string

	Directories []string
	Locked      bool
	Lazy        bool

	Chdir                       string
	AllowAllSymlinkDestinations bool
}

func (o *SyncOptions) LockFileExists() bool {
	if _, err := os.Stat(o.LockFile); err != nil {
		return false
	}
	return true
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
	cmd.Flags().StringSliceVarP(&o.Files, "file", "f", []string{defaultConfigName}, "Set configuration file")
	cmd.Flags().StringVar(&o.LockFile, "lock-file", defaultLockName, "Set lock file")

	cmd.Flags().StringSliceVarP(&o.Directories, "directory", "d", nil, "Sync specific directory (format: dir/sub-dir[=local-dir])")
	cmd.Flags().BoolVarP(&o.Locked, "locked", "l", false, "Consult lock file to pull exact references (e.g. use git sha instead of branch name)")
	cmd.Flags().BoolVar(&o.Lazy, "lazy", true, "Set to 'false' it ignores the 'lazy' flag in the directory content configuration")

	cmd.Flags().StringVar(&o.Chdir, "chdir", "", "Set current directory for process")
	cmd.Flags().BoolVar(&o.AllowAllSymlinkDestinations, "dangerous-allow-all-symlink-destinations", false, "Symlinks to all destinations are allowed")

	return cmd
}

func (o *SyncOptions) Run() error {
	if len(o.Chdir) > 0 {
		err := os.Chdir(o.Chdir)
		if err != nil {
			return fmt.Errorf("Running chdir: %s", err)
		}
	}

	conf, secrets, configMaps, err := ctlconf.NewConfigFromFiles(o.Files)
	if err != nil {
		return o.configReadHintErrMsg(err, o.Files)
	}

	dirs, err := o.directories()
	if err != nil {
		return err
	}

	usesLocalDir, err := o.applyUseDirectories(&conf, dirs)
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

	existingLockConfig, err := ctlconf.NewLockConfigFromFile(o.LockFile)
	if err != nil && o.Locked {
		return err
	}

	// If syncing against a lock file, apply lock information
	// on top of existing config
	if o.Locked {
		err = conf.Lock(existingLockConfig)
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
	maxCacheableContentSize := os.Getenv("VENDIR_MAX_CACHE_SIZE")
	if maxCacheableContentSize == "" {
		maxCacheableContentSize = "1Mi"
	}

	cache, err := ctlcache.NewCache(os.Getenv("VENDIR_CACHE_DIR"), maxCacheableContentSize)
	if err != nil {
		return fmt.Errorf("Unable to create cache: %s", err)
	}
	syncOpts := ctldir.SyncOpts{
		RefFetcher:     ctldir.NewNamedRefFetcher(secrets, configMaps),
		GithubAPIToken: os.Getenv("VENDIR_GITHUB_API_TOKEN"),
		HelmBinary:     os.Getenv("VENDIR_HELM_BINARY"),
		Cache:          cache,
		Lazy:           o.Lazy,
	}
	newLockConfig := ctlconf.NewLockConfig()
	for _, dirConf := range conf.Directories {
		// error safe to ignore, since lock file might not exist
		dirExistingLockConf, _ := existingLockConfig.FindDirectory(dirConf.Path)
		directory := ctldir.NewDirectory(dirConf, dirExistingLockConf, o.ui)
		dirLockConf, err := directory.Sync(syncOpts)
		if err != nil {
			return fmt.Errorf("Syncing directory '%s': %s", dirConf.Path, err)
		}
		if !o.AllowAllSymlinkDestinations {
			err = ctldir.ValidateSymlinks(dirConf.Path)
			if err != nil {
				return err
			}
		}

		newLockConfig.Directories = append(newLockConfig.Directories, dirLockConf)
	}

	// Update only selected directories in lock file
	if len(dirs) > 0 {
		if o.LockFileExists() {
			existingLockConfig, err := ctlconf.NewLockConfigFromFile(o.LockFile)
			if err != nil {
				return err
			}
			err = existingLockConfig.Merge(newLockConfig)
			if err != nil {
				return err
			}

			newLockConfig = existingLockConfig
		}
	}

	// Clean old dirs from final output dir
	// Iterate over directory paths
	for _, dir := range newLockConfig.Directories {
		// Walk through file system dirs and match with lock file config
		err := filepath.WalkDir(dir.Path, func(filePath string, info os.DirEntry, err error) error {
			if err != nil {
				// dirs might be unreadable due to restrictive file permissions set in vendir.yml
				// therefore ignore errors on traversing directories
				return nil
			}
			for _, dir := range newLockConfig.Directories {
				for _, content := range dir.Contents {
					// if file system path found in config file then skip
					if filepath.Join(dir.Path, content.Path) == filePath {
						return filepath.SkipDir
					}
					// if file system path is a parent directory of config file then continue iterating
					if strings.HasPrefix(filepath.Join(dir.Path, content.Path), filePath+"/") {
						return nil
					}
				}
			}
			// if file system path not found in config file then delete
			err = os.RemoveAll(filePath)
			if err != nil {
				return err
			}
			return filepath.SkipDir
		})
		if err != nil {
			return fmt.Errorf("Error while cleaning old directories: %s", err)
		}
	}

	newLockConfigBs, err := newLockConfig.AsBytes()
	if err != nil {
		return err
	}

	o.ui.PrintLinef("Lock config")
	o.ui.PrintBlock(newLockConfigBs)

	if usesLocalDir {
		o.ui.PrintLinef("Lock config is not saved to '%s' due to command line overrides", o.LockFile)
		return nil
	}

	return newLockConfig.WriteToFile(o.LockFile)
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

func (o *SyncOptions) applyUseDirectories(conf *ctlconf.Config, dirs []dirOverride) (bool, error) {
	usesLocalDir := false

	for _, dir := range dirs {
		if len(dir.LocalDir) == 0 {
			continue
		}
		usesLocalDir = true

		err := conf.UseDirectory(dir.Path, dir.LocalDir)
		if err != nil {
			return false, fmt.Errorf("Overriding '%s' with local directory: %s", dir.Path, err)
		}
	}
	return usesLocalDir, nil
}

func (*SyncOptions) configReadHintErrMsg(origErr error, paths []string) error {
	if len(paths) != 1 {
		return origErr
	}
	path := paths[0]

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) && path == defaultConfigName {
			hintMsg := "(hint: Did you name your configuration file something different than 'vendir.yml', e.g. wrong extension?)"
			return fmt.Errorf("%s %s", origErr, hintMsg)
		}
	}
	return origErr
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
	out, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("Expanding user home directory: %s", err)
	}
	return strings.TrimSpace(string(out)), nil
}
