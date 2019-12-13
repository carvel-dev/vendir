package cmd

import (
	"github.com/cppforlife/go-cli-ui/ui"
	ctlconf "github.com/k14s/vendir/pkg/vendir/config"
	ctldir "github.com/k14s/vendir/pkg/vendir/directory"
	"github.com/spf13/cobra"
)

type SyncOptions struct {
	ui ui.UI
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
	return cmd
}

func (o *SyncOptions) Run() error {
	conf, err := ctlconf.NewConfigFromFile("vendir.yml")
	if err != nil {
		return err
	}

	lockConfig := ctlconf.NewLockConfig()

	for _, dirConf := range conf.Directories {
		dirLockConf, err := ctldir.NewDirectory(dirConf).Sync()
		if err != nil {
			return err
		}

		lockConfig.Directories = append(lockConfig.Directories, dirLockConf)
	}

	return lockConfig.WriteToFile("vendir.lock.yml")
}
