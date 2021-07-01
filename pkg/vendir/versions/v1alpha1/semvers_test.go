// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"reflect"
	"testing"

	versions "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions/v1alpha1"
)

func TestSemverOrder(t *testing.T) {
	result := versions.NewRelaxedSemversNoErr([]string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-pre.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"v2.0.0", // prefixed with v
		"0.0.1-rc.0",
	}).Sorted().All()

	expectedOrder := []string{
		"0.0.1-pre.1",
		"0.0.1-pre.10",
		"0.0.1-rc.0",
		"0.1.0",
		"2.0.0-10",
		"2.0.0-10+meta.10",
		"2.0.0",
		"v2.0.0",
	}

	if !reflect.DeepEqual(result, expectedOrder) {
		t.Fatalf("Expected result '%#v' to equal '%#v'", result, expectedOrder)
	}
}

func TestSemverFilter(t *testing.T) {
	result, err := versions.NewRelaxedSemversNoErr([]string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-pre.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"0.0.1-rc.0",
	}).Sorted().FilterConstraints(">0.0.5 <5.0.0")
	if err != nil {
		t.Fatalf("Expected filtering to succeed: %s", err)
	}

	expectedOrder := []string{
		"0.1.0",
		"2.0.0-10",
		"2.0.0-10+meta.10", // prerelease is included
		"2.0.0",
	}

	if !reflect.DeepEqual(result.All(), expectedOrder) {
		t.Fatalf("Expected result '%#v' to equal '%#v'", result.All(), expectedOrder)
	}
}

func TestSemverWithoutPrereleases(t *testing.T) {
	result := versions.NewRelaxedSemversNoErr([]string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-pre.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"0.0.1-rc.0",
	}).FilterPrereleases(nil)

	expectedOrder := []string{
		"0.1.0",
		"2.0.0",
	}

	if !reflect.DeepEqual(result.All(), expectedOrder) {
		t.Fatalf("Expected result '%#v' to equal '%#v'", result.All(), expectedOrder)
	}
}

func TestSemverWithPrereleases(t *testing.T) {
	preConf := &versions.VersionSelectionSemverPrereleases{}

	result := versions.NewRelaxedSemversNoErr([]string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-pre.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"0.0.1-rc.0",
	}).FilterPrereleases(preConf)

	expectedOrder := []string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-pre.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"0.0.1-rc.0",
	}

	if !reflect.DeepEqual(result.All(), expectedOrder) {
		t.Fatalf("Expected result '%#v' to equal '%#v'", result.All(), expectedOrder)
	}
}

func TestSemverWithPrereleaseIdentifiers(t *testing.T) {
	preConf := &versions.VersionSelectionSemverPrereleases{Identifiers: []string{"alpha", "rc"}}

	result := versions.NewRelaxedSemversNoErr([]string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-alpha.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"0.0.1-rc.0",
	}).Sorted().FilterPrereleases(preConf)

	expectedOrder := []string{
		"0.0.1-alpha.1",
		"0.0.1-rc.0",
		"0.1.0",
		"2.0.0",
	}

	if !reflect.DeepEqual(result.All(), expectedOrder) {
		t.Fatalf("Expected result '%#v' to equal '%#v'", result.All(), expectedOrder)
	}
}

func TestSemverWithBuildMetadata(t *testing.T) {
	result := versions.NewRelaxedSemversNoErr([]string{
		"1.0.0",
		"1.0.0+1",
		"1.0.0+2",
		"1.0.0+ab1",
		"1.0.0+z1",
		"1.0.0+ab1.foo",
		"1.0.0-pre+foo",
		"1.1.0",
		"1.1.0+aaaa",
		"2.0.0",
	}).Sorted().All()

	expectedOrder := []string{
		"1.0.0-pre+foo",
		"1.0.0",
		"1.0.0+1",
		"1.0.0+2",
		"1.0.0+ab1",
		"1.0.0+ab1.foo",
		"1.0.0+z1",
		"1.1.0",
		"1.1.0+aaaa",
		"2.0.0",
	}

	if !reflect.DeepEqual(result, expectedOrder) {
		t.Fatalf("Expected result '%#v' to equal '%#v'", result, expectedOrder)
	}
}

func TestSemverWithBuildMetadataAndConstraint(t *testing.T) {
	result, err := versions.NewRelaxedSemversNoErr([]string{
		"1.0.0",
		"1.0.0+1",
		"1.0.0+2",
		"1.0.0+ab1",
		"1.0.0+z1",
		"1.0.0+ab1.foo",
		"1.0.0-pre+foo",
	}).Sorted().FilterConstraints(">1.0.0+ab1.foo")

	if err != nil {
		t.Fatalf("Received error when filtering: %v", err)
	}

	expectedOrder := []string{
		"1.0.0+z1",
	}

	if !reflect.DeepEqual(result.All(), expectedOrder) {
		t.Fatalf("Expected result '%#v' to equal '%#v'", result.All(), expectedOrder)
	}
}
