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

var componentTags = []string{"sf/v1.2.0", "sf/v1.10.0", "sf/v1.11.0-rc.1", "sf/v2.0.0", "other/v1.99.0", "v1.99.0"}

func TestSemverPrefixParsing(t *testing.T) {
	t.Run("empty prefix parses the same versions as no prefix", func(t *testing.T) {
		assert.Equal(t,
			versions.NewRelaxedSemversNoErr(componentTags).Sorted().All(),
			versions.NewPrefixedRelaxedSemversNoErr(componentTags, "").Sorted().All())
	})

	t.Run("prefix match is literal and case sensitive", func(t *testing.T) {
		assert.Empty(t, versions.NewPrefixedRelaxedSemversNoErr(componentTags, "SF/").All())
		assert.Empty(t, versions.NewPrefixedRelaxedSemversNoErr([]string{"sf.v1.0.0", "sfv1.0.0", "xsf/v1.0.0"}, "sf/").All())
	})

	t.Run("prefix is removed once and then optional v is handled", func(t *testing.T) {
		result := versions.NewPrefixedRelaxedSemversNoErr([]string{
			"sf/1.2.3",
			"sf/v1.2.4",
			"sf/sf/v1.2.5", // remainder "sf/v1.2.5" is not a version
		}, "sf/").Sorted().All()
		assert.Equal(t, []string{"sf/1.2.3", "sf/v1.2.4"}, result)

		result = versions.NewPrefixedRelaxedSemversNoErr([]string{"sf-v1.2.3", "sf-1.2.3", "sf/v1.2.3"}, "sf-v").All()
		assert.Equal(t, []string{"sf-v1.2.3"}, result)
	})

	t.Run("orders remainder numerically and keeps original tags", func(t *testing.T) {
		result := versions.NewPrefixedRelaxedSemversNoErr([]string{
			"sf/v1.10.0",
			"sf/v1.9.0",
			"sf/v1.10.0-rc.1",
			"sf/v1.2.0+build.7",
		}, "sf/").Sorted().All()
		assert.Equal(t, []string{"sf/v1.2.0+build.7", "sf/v1.9.0", "sf/v1.10.0-rc.1", "sf/v1.10.0"}, result)
	})

	t.Run("empty or malformed remainders are ignored like other invalid versions", func(t *testing.T) {
		assert.Empty(t, versions.NewPrefixedRelaxedSemversNoErr([]string{"sf/", "sf/v", "sf/latest", "sf/v1", "sf/1.2"}, "sf/").All())
	})
}

func TestSemverPrefixSelection(t *testing.T) {
	selection := func(prefix string, constraints string, prereleases *v1alpha1.VersionSelectionSemverPrereleases) v1alpha1.VersionSelection {
		return v1alpha1.VersionSelection{Semver: &v1alpha1.VersionSelectionSemver{
			Prefix: prefix, Constraints: constraints, Prereleases: prereleases,
		}}
	}

	t.Run("omitted prefix keeps existing behavior for plain and v-prefixed versions", func(t *testing.T) {
		highest, err := versions.HighestConstrainedVersion([]string{"1.0.0", "v1.5.0", "1.2.0", "sf/v9.0.0"}, selection("", "", nil))
		require.NoError(t, err)
		assert.Equal(t, "v1.5.0", highest)

		highest, err = versions.HighestConstrainedVersion(componentTags, selection("", "", nil))
		require.NoError(t, err)
		assert.Equal(t, "v1.99.0", highest)
	})

	t.Run("selects highest constrained version among prefixed tags only", func(t *testing.T) {
		highest, err := versions.HighestConstrainedVersion(componentTags, selection("sf/", ">=1.0.0 <2.0.0", nil))
		require.NoError(t, err)
		assert.Equal(t, "sf/v1.10.0", highest)
	})

	t.Run("prereleases are excluded by default and can be opted in", func(t *testing.T) {
		highest, err := versions.HighestConstrainedVersion(componentTags, selection("sf/", "<2.0.0", nil))
		require.NoError(t, err)
		assert.Equal(t, "sf/v1.10.0", highest)

		highest, err = versions.HighestConstrainedVersion(componentTags, selection("sf/", "<2.0.0", &v1alpha1.VersionSelectionSemverPrereleases{}))
		require.NoError(t, err)
		assert.Equal(t, "sf/v1.11.0-rc.1", highest)

		highest, err = versions.HighestConstrainedVersion(componentTags, selection("sf/", "<2.0.0", &v1alpha1.VersionSelectionSemverPrereleases{Identifiers: []string{"beta"}}))
		require.NoError(t, err)
		assert.Equal(t, "sf/v1.10.0", highest)
	})

	t.Run("original tag with build metadata is preserved", func(t *testing.T) {
		highest, err := versions.HighestConstrainedVersion([]string{"sf/v1.0.0", "sf/v1.1.0+build.7", "sf/v1.0.5"}, selection("sf/", "", nil))
		require.NoError(t, err)
		assert.Equal(t, "sf/v1.1.0+build.7", highest)
	})

	t.Run("no qualifying candidates uses the existing error", func(t *testing.T) {
		_, err := versions.HighestConstrainedVersion(componentTags, selection("SF/", "", nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Expected to find at least one version, but none matched")

		_, err = versions.HighestConstrainedVersion([]string{"sf/", "sf/latest", "v1.0.0"}, selection("sf/", "", nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Expected to find at least one version, but none matched")
	})

	t.Run("error details distinguish a prefix that matches nothing", func(t *testing.T) {
		_, err := versions.HighestConstrainedVersion(componentTags, selection("SF/", "", nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "(details: with-prefix=0 -> all=0 -> ")

		_, err = versions.HighestConstrainedVersion([]string{"sf/", "sf/latest"}, selection("sf/", "", nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "(details: with-prefix=2 -> all=0 -> ")

		_, err = versions.HighestConstrainedVersion(componentTags, selection("", ">9.0.0", nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "(details: all=1 -> ")
	})
}
