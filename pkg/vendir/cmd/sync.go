package cmd

import (
	"fmt"
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

	File         string
	Directories  []string
	UseDirectory []string
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
	cmd.Flags().StringSliceVarP(&o.Directories, "directory", "d", nil, "Only sync specific directory (format: dir/sub-dir[=local-dir])")
	cmd.Flags().StringSliceVar(&o.UseDirectory, "use-directory", nil, "Set directory configuration (format: dir/sub-dir=local-dir)")
	return cmd
}

func (o *SyncOptions) Run() error {
	conf, err := ctlconf.NewConfigFromFile(o.File)
	if err != nil {
		return err
	}

	dirs, err := o.markedDirectories()
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

	shouldWriteLockConfig := (o.File == defaultConfigName) && len(dirs) == 0
	lockConfig := ctlconf.NewLockConfig()

	for _, dirConf := range conf.Directories {
		dirLockConf, err := ctldir.NewDirectory(dirConf, o.ui).Sync()
		if err != nil {
			return err
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

func (o *SyncOptions) markedDirectories() ([]dirOverride, error) {
	var dirs []dirOverride

	for _, val := range o.Directories {
		pieces := strings.SplitN(val, "=", 2)
		if len(pieces) == 1 {
			dirs = append(dirs, dirOverride{Path: pieces[0]})
		} else {
			dirs = append(dirs, dirOverride{Path: pieces[0], LocalDir: pieces[1]})
		}
	}

	for _, val := range o.UseDirectory {
		pieces := strings.SplitN(val, "=", 2)
		if len(pieces) != 2 {
			return nil, fmt.Errorf("Expected '--use-directory' flag value '%s' to be in format 'dir/sub-dir=local-dir'", val)
		}

		dirs = append(dirs, dirOverride{Path: pieces[0], LocalDir: pieces[1]})
	}

	return dirs, nil
}

func (o *SyncOptions) applyUseDirectories(conf *ctlconf.Config, dirs []dirOverride) error {
	for _, dir := range dirs {
		if len(dir.LocalDir) == 0 {
			continue
		}
		err := conf.UseDirectory(dir.Path, dir.LocalDir)
		if err != nil {
			return fmt.Errorf("Overriding '%s' with local directory", dir.Path)
		}
	}
	return nil
}
