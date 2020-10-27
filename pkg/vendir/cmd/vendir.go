// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"io"

	"github.com/cppforlife/cobrautil"
	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/k14s/vendir/pkg/vendir/version"
	"github.com/spf13/cobra"
)

type VendirOptions struct {
	ui *ui.ConfUI

	UIFlags UIFlags
}

func NewVendirOptions(ui *ui.ConfUI) *VendirOptions {
	return &VendirOptions{ui: ui}
}

func NewDefaultVendirCmd(ui *ui.ConfUI) *cobra.Command {
	return NewVendirCmd(NewVendirOptions(ui))
}

func NewVendirCmd(o *VendirOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "vendir",
		Short:             "vendir allows to declaratively state what should be in a directory",
		SilenceErrors:     true,
		SilenceUsage:      true,
		DisableAutoGenTag: true,
		Version:           version.Version,
	}

	// TODO bash completion
	cmd.SetOutput(uiBlockWriter{o.ui}) // setting output for cmd.Help()

	o.UIFlags.Set(cmd)

	cmd.AddCommand(NewSyncCmd(NewSyncOptions(o.ui)))
	cmd.AddCommand(NewVersionCmd(NewVersionOptions(o.ui)))

	// Last one runs first
	cobrautil.VisitCommands(cmd, cobrautil.ReconfigureCmdWithSubcmd)
	cobrautil.VisitCommands(cmd, cobrautil.ReconfigureLeafCmd)

	cobrautil.VisitCommands(cmd, cobrautil.WrapRunEForCmd(func(*cobra.Command, []string) error {
		o.UIFlags.ConfigureUI(o.ui)
		return nil
	}))

	cobrautil.VisitCommands(cmd, cobrautil.WrapRunEForCmd(cobrautil.ResolveFlagsForCmd))

	return cmd
}

type uiBlockWriter struct {
	ui ui.UI
}

var _ io.Writer = uiBlockWriter{}

func (w uiBlockWriter) Write(p []byte) (n int, err error) {
	w.ui.PrintBlock(p)
	return len(p), nil
}
