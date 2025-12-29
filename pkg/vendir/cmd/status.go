package cmd

import (
	"fmt"
	"os"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/spf13/cobra"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	ctldir "carvel.dev/vendir/pkg/vendir/directory"
	ctlcache "carvel.dev/vendir/pkg/vendir/fetch/cache"
	ctlstatus "carvel.dev/vendir/pkg/vendir/status"
)

type StatusOptions struct {
	ui ui.UI

	Files    []string
	LockFile string

	Chdir    string
	ExitCode bool
}

func NewStatusOptions(ui ui.UI) *StatusOptions {
	return &StatusOptions{ui: ui}
}

func NewStatusCmd(o *StatusOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check local repositories status",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
	}

	cmd.Flags().StringSliceVarP(&o.Files, "file", "f", []string{defaultConfigName}, "Set configuration file")
	cmd.Flags().StringVar(&o.LockFile, "lock-file", defaultLockName, "Set lock file")
	cmd.Flags().StringVar(&o.Chdir, "chdir", "", "Set current directory for process")
	cmd.Flags().BoolVar(&o.ExitCode, "exit-code", false, "Set to 'true', it exits with a non-0 code if any subproject is not clean")

	return cmd
}

func (o *StatusOptions) Run() error {
	if len(o.Chdir) > 0 {
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

	status, err := fullStatus(conf, syncOpts, existingLockConfig, o.ui)
	if err != nil {
		return err
	}

	o.ui.PrintBlock([]byte("---------------\n\n"))
	o.ui.PrintBlock([]byte(status.String()))

	return nil
}

func fullStatus(
	conf ctlconf.Config,
	syncOpts ctldir.SyncOpts,
	existingLockConfig ctlconf.LockConfig,
	ui ui.UI,
) (ctlstatus.StatusList, error) {
	status := ctlstatus.StatusList{}
	for _, dirConf := range conf.Directories {
		dirExistingLockConf, _ := existingLockConfig.FindDirectory(dirConf.Path)
		directory := ctldir.NewDirectory(dirConf, dirExistingLockConf, ui)

		dirStatus, err := directory.Status(syncOpts)
		if err != nil {
			return nil, fmt.Errorf("Reading directory '%s': %s", dirConf.Path, err)
		}

		status = append(status, dirStatus...)
	}

	return status, nil
}
