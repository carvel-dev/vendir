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
	ui           ui.UI
	File         string
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
	cmd.Flags().StringSliceVar(&o.UseDirectory, "use-directory", nil, "Set directory configuration (format: dir/sub-dir=local-dir)")
	return cmd
}

func (o *SyncOptions) Run() error {
	conf, err := ctlconf.NewConfigFromFile(o.File)
	if err != nil {
		return err
	}

	err = o.applyUseDirectories(&conf)
	if err != nil {
		return err
	}

	shouldWriteLockConfig := (o.File == defaultConfigName) && len(o.UseDirectory) == 0
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

func (o *SyncOptions) applyUseDirectories(conf *ctlconf.Config) error {
	for _, val := range o.UseDirectory {
		pieces := strings.SplitN(val, "=", 2)
		if len(pieces) != 2 {
			return fmt.Errorf("Expected '--use-directory' flag value '%s' to be in format 'dir/sub-dir=local-dir'", val)
		}

		err := conf.UseDirectory(pieces[0], pieces[1])
		if err != nil {
			return fmt.Errorf("Applying '--use-directory' flag value '%s' to config", val)
		}
	}

	return nil
}
