// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"
	"sort"
	"strings"

	semver "github.com/blang/semver/v4"
)

type Semvers struct {
	versions []semverWrap
}

type semverWrap struct {
	semver.Version
	Original string
}

func NewSemvers(versions []string) Semvers {
	var parsedVersions []semverWrap

	for _, vStr := range versions {
		ver, err := semver.Parse(vStr)
		if err == nil {
			parsedVersions = append(parsedVersions, semverWrap{Version: ver, Original: vStr})
		} else if strings.HasPrefix(vStr, "v") {
			ver, err := semver.Parse(strings.TrimPrefix(vStr, "v"))
			if err == nil {
				parsedVersions = append(parsedVersions, semverWrap{Version: ver, Original: vStr})
			}
		} else {
			// Ignore non-parseable versions
		}
	}

	return Semvers{parsedVersions}
}

func (v Semvers) Sorted() Semvers {
	var versions []semverWrap

	for _, ver := range v.versions {
		versions = append(versions, ver)
	}

	sort.SliceStable(versions, func(i, j int) bool {
		return versions[i].Version.LT(versions[j].Version)
	})

	return Semvers{versions}
}

func (v Semvers) FilterConstraints(constraintList string) (Semvers, error) {
	constraints, err := semver.ParseRange(constraintList)
	if err != nil {
		return Semvers{}, fmt.Errorf("Parsing version constraint '%s': %s", constraintList, err)
	}

	var matchingVersions []semverWrap

	for _, ver := range v.versions {
		if constraints(ver.Version) {
			matchingVersions = append(matchingVersions, ver)
		}
	}

	return Semvers{matchingVersions}, nil
}

func (v Semvers) FilterPrereleases(prereleases *VersionSelectionSemverPrereleases) Semvers {
	if prereleases == nil {
		// Exclude all prereleases
		var result []semverWrap
		for _, ver := range v.versions {
			if len(ver.Version.Pre) == 0 {
				result = append(result, ver)
			}
		}
		return Semvers{result}
	}

	preIdentifiersAsMap := prereleases.IdentifiersAsMap()

	var result []semverWrap
	for _, ver := range v.versions {
		if len(ver.Version.Pre) == 0 || v.shouldKeepPrerelease(ver.Version, preIdentifiersAsMap) {
			result = append(result, ver)
		}
	}
	return Semvers{result}
}

func (Semvers) shouldKeepPrerelease(ver semver.Version, preIdentifiersAsMap map[string]struct{}) bool {
	if len(preIdentifiersAsMap) == 0 {
		return true
	}
	for _, prePart := range ver.Pre {
		if len(prePart.VersionStr) > 0 {
			if _, found := preIdentifiersAsMap[prePart.VersionStr]; found {
				return true
			}
		}
	}
	return false
}

func (v Semvers) Highest() (string, bool) {
	v = v.Sorted()

	if len(v.versions) == 0 {
		return "", false
	}

	return v.versions[len(v.versions)-1].Original, true
}

func (v Semvers) All() []string {
	var verStrs []string
	for _, ver := range v.versions {
		verStrs = append(verStrs, ver.Original)
	}
	return verStrs
}
