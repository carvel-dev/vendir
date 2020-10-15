// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	uitable "github.com/cppforlife/go-cli-ui/ui/table"
	ctldir "github.com/k14s/vendir/pkg/vendir/directory"
	"github.com/spf13/cobra"
)

type SortVersionsOptions struct {
	ui ui.UI

	Constraints []string
	Versions    []string
}

func NewSortVersionsOptions(ui ui.UI) *SortVersionsOptions {
	return &SortVersionsOptions{ui: ui}
}

func NewSortVersionsCmd(o *SortVersionsOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sort-versions",
		Short: "Sort versions",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
	}
	cmd.Flags().StringSliceVarP(&o.Constraints, "constraint", "c", nil, "Constraints (e.g. '>=v1.0, <v2.0')")
	cmd.Flags().StringSliceVarP(&o.Versions, "version", "v", nil, "List of versions")
	return cmd
}

func (o *SortVersionsOptions) Run() error {
	allVers := ctldir.NewVersions(o.versions()).Sorted()

	if len(o.Constraints) > 0 {
		var err error

		constraints := strings.Join(o.Constraints, ", ")
		allVers, err = allVers.Filtered(constraints)
		if err != nil {
			return err
		}
	}

	table := uitable.Table{
		Title:           "Versions",
		FillFirstColumn: true,
		Header: []uitable.Header{
			uitable.NewHeader("Version"),
		},
	}

	for _, ver := range allVers.All() {
		table.Rows = append(table.Rows, []uitable.Value{
			uitable.NewValueString(ver),
		})
	}

	o.ui.PrintTable(table)

	highestVer, found := allVers.Highest()
	if found {
		o.ui.PrintLinef("Highest version: %s", highestVer)
	}

	return nil
}

func (o *SortVersionsOptions) versions() []string {
	var vers []string
	for _, ver := range o.Versions {
		vers = append(vers, strings.Fields(ver)...)
	}
	return vers
}
