// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package versions_test

import (
	"strings"
	"testing"

	versions "carvel.dev/vendir/pkg/vendir/versions"
	"carvel.dev/vendir/pkg/vendir/versions/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	require.Equal(t, []string{
		"0.0.1-pre.1",
		"0.0.1-pre.10",
		"0.0.1-rc.0",
		"0.1.0",
		"2.0.0-10",
		"2.0.0-10+meta.10",
		"2.0.0",
		"v2.0.0",
	}, result)
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
	require.NoError(t, err)

	require.Equal(t, []string{
		"0.1.0",
		"2.0.0-10",
		"2.0.0-10+meta.10", // prerelease is included
		"2.0.0",
	}, result.All())
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

	require.Equal(t, []string{
		"0.1.0",
		"2.0.0",
	}, result.All())
}

func TestSemverWithPrereleases(t *testing.T) {
	preConf := &v1alpha1.VersionSelectionSemverPrereleases{}

	result := versions.NewRelaxedSemversNoErr([]string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-pre.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"0.0.1-rc.0",
	}).FilterPrereleases(preConf)

	require.Equal(t, []string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-pre.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"0.0.1-rc.0",
	}, result.All())
}

func TestSemverWithPrereleaseIdentifiers(t *testing.T) {
	preConf := &v1alpha1.VersionSelectionSemverPrereleases{Identifiers: []string{"alpha", "rc"}}

	result := versions.NewRelaxedSemversNoErr([]string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-alpha.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"0.0.1-rc.0",
	}).Sorted().FilterPrereleases(preConf)

	require.Equal(t, []string{
		"0.0.1-alpha.1",
		"0.0.1-rc.0",
		"0.1.0",
		"2.0.0",
	}, result.All())
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

	require.Equal(t, []string{
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
	}, result)
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
	require.NoError(t, err)

	require.Equal(t, []string{"1.0.0+z1"}, result.All())
}

func TestHighestVersionWithConstraints(t *testing.T) {
	vs := []string{
		"2.0.3",
		"0.0.1",
		"0.3.1",
		"0.3.0",
		"2.0.0",
		"2.3.0",
		"0.0.1",
	}

	constraint := func(verStr string) bool {
		return !strings.Contains(verStr, "3")
	}

	answer, err := versions.HighestConstrainedVersionWithAdditionalConstraints(vs,
		v1alpha1.VersionSelection{Semver: &v1alpha1.VersionSelectionSemver{Constraints: ">0.1.0"}},
		[]versions.ConstraintCallback{{Constraint: constraint, Name: "myConstraint"}})

	require.NoError(t, err)
	assert.Equal(t, "2.0.0", answer)

	_, err = versions.HighestConstrainedVersionWithAdditionalConstraints(vs,
		v1alpha1.VersionSelection{Semver: &v1alpha1.VersionSelectionSemver{Constraints: ">2.1.0"}},
		[]versions.ConstraintCallback{{Constraint: constraint, Name: "myConstraint"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "myConstraint")
}

func TestHighestPrefersGAOverRCInBuildMetadata(t *testing.T) {
	// Bug 1: RC version was considered higher than GA/RTM
	t.Run("PreRelease_ShouldBeLowerThan_GA_EvenWithMetadata", func(t *testing.T) {
		versionsList := []string{"3.4.0+v1.33-rc.2", "3.4.0+v1.33"}
		result := versions.NewRelaxedSemversNoErr(versionsList).Sorted().All()

		// The first element should be the RC, the last (highest) should be GA
		expected := []string{"3.4.0+v1.33-rc.2", "3.4.0+v1.33"}
		require.Equal(t, expected, result, "GA version must be sorted higher than RC version")
	})
}

func TestVersionSortHandlesNumericBuildMetadata(t *testing.T) {
	// Bug 2: Lexicographical sorting instead of Natural sorting (9 > 10)
	t.Run("BuildMetadata_ShouldUseNaturalSort", func(t *testing.T) {
		versionsList := []string{"v1.2.3+vmware.10-fips", "v1.2.3+vmware.9-fips"}
		result := versions.NewRelaxedSemversNoErr(versionsList).Sorted().All()

		// 'vmware.9' should come before 'vmware.10'
		expected := []string{"v1.2.3+vmware.9-fips", "v1.2.3+vmware.10-fips"}
		require.Equal(t, expected, result, "Version with metadata .10 should be higher than .9")
	})
}
