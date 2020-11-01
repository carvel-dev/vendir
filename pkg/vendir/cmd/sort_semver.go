// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	uitable "github.com/cppforlife/go-cli-ui/ui/table"
	ctlconf "github.com/k14s/vendir/pkg/vendir/config"
	ctlver "github.com/k14s/vendir/pkg/vendir/versions"
	"github.com/spf13/cobra"
)

type SortSemverOptions struct {
	ui ui.UI

	Constraints           []string
	Versions              []string
	Prerelease            bool
	PrereleaseIdentifiers []string
}

func NewSortSemverOptions(ui ui.UI) *SortSemverOptions {
	return &SortSemverOptions{ui: ui}
}

func NewSortSemverCmd(o *SortSemverOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sort-semver",
		Short: "Sort semver versions",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
	}
	cmd.Flags().StringSliceVarP(&o.Constraints, "constraint", "c", nil, "Constraints (e.g. '>=v1.0, <v2.0')")
	cmd.Flags().StringSliceVarP(&o.Versions, "version", "v", nil, "List of versions")
	cmd.Flags().BoolVar(&o.Prerelease, "prerelease", false, "Include prerelease versions")
	cmd.Flags().StringSliceVar(&o.PrereleaseIdentifiers, "prerelease-identifier", nil, "Include prerelease version identifier")
	return cmd
}

func (o *SortSemverOptions) Run() error {
	allVers := ctlver.NewSemvers(o.versions()).Sorted()

	if len(o.Constraints) > 0 {
		var err error

		constraints := strings.Join(o.Constraints, ", ")
		allVers, err = allVers.FilterConstraints(constraints)
		if err != nil {
			return err
		}
	}

	allVers = allVers.FilterPrereleases(o.prereleaseConf())

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

func (o *SortSemverOptions) versions() []string {
	var vers []string
	for _, ver := range o.Versions {
		vers = append(vers, strings.Fields(ver)...)
	}
	return vers
}

func (o *SortSemverOptions) prereleaseConf() *ctlconf.VersionSelectionSemverPrereleases {
	if o.Prerelease || len(o.PrereleaseIdentifiers) > 0 {
		result := &ctlconf.VersionSelectionSemverPrereleases{}
		result.Identifiers = o.PrereleaseIdentifiers
		return result
	}
	return nil
}
