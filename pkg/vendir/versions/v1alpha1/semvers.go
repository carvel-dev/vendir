// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	semver "github.com/blang/semver/v4"
)

const (
	lessThan    = -1
	greaterThan = 1
	equalTo     = 0
)

type Semvers struct {
	versions []SemverWrap
}

type SemverWrap struct {
	version  semver.Version
	Original string
}

func (sw SemverWrap) Compare(subj SemverWrap) int {
	if result := sw.version.Compare(subj.version); result != 0 {
		return result
	}

	return newBuildMeta(sw.version.Build).compare(newBuildMeta(subj.version.Build))
}

func NewSemver(version string) (SemverWrap, error) {
	parsedVersion, err := semver.Parse(version)
	if err != nil {
		return SemverWrap{}, err
	}

	return SemverWrap{parsedVersion, version}, nil
}

func NewRelaxedSemver(version string) (SemverWrap, error) {
	parsableVersion := version
	if strings.HasPrefix(version, "v") {
		parsableVersion = strings.TrimPrefix(version, "v")
	}

	parsedVersion, err := semver.Parse(parsableVersion)
	if err != nil {
		return SemverWrap{}, err
	}

	return SemverWrap{parsedVersion, version}, nil
}

func NewRelaxedSemversNoErr(versions []string) Semvers {
	var parsedVersions []SemverWrap

	for _, vStr := range versions {
		ver, err := NewRelaxedSemver(vStr)
		if err != nil {
			continue
		}
		parsedVersions = append(parsedVersions, ver)
	}

	return Semvers{parsedVersions}
}

func (v Semvers) Sorted() Semvers {
	var versions []SemverWrap

	for _, ver := range v.versions {
		versions = append(versions, ver)
	}

	sort.SliceStable(versions, func(i, j int) bool {
		return versions[i].Compare(versions[j]) == lessThan
	})

	return Semvers{versions}
}

func (v Semvers) FilterConstraints(constraintList string) (Semvers, error) {
	constraints, err := semver.ParseRange(constraintList)
	if err != nil {
		return Semvers{}, fmt.Errorf("Parsing version constraint '%s': %s", constraintList, err)
	}

	var matchingVersions []SemverWrap

	for _, ver := range v.versions {
		if constraints(ver.version) {
			matchingVersions = append(matchingVersions, ver)
		}
	}

	return Semvers{matchingVersions}, nil
}

func (v Semvers) FilterPrereleases(prereleases *VersionSelectionSemverPrereleases) Semvers {
	if prereleases == nil {
		// Exclude all prereleases
		var result []SemverWrap
		for _, ver := range v.versions {
			if len(ver.version.Pre) == 0 {
				result = append(result, ver)
			}
		}
		return Semvers{result}
	}

	preIdentifiersAsMap := prereleases.IdentifiersAsMap()

	var result []SemverWrap
	for _, ver := range v.versions {
		if len(ver.version.Pre) == 0 || v.shouldKeepPrerelease(ver.version, preIdentifiersAsMap) {
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

type buildMeta struct {
	parts []buildMetaPart
}

// Since this method is private to the package,
// we are not doing any validations of characters
// here and are instead relying on the semver
// library parsing
func newBuildMeta(metaParts []string) buildMeta {
	parts := make([]buildMetaPart, len(metaParts))
	for i, str := range metaParts {
		parts[i] = newBuildMetaPart(str)
	}
	return buildMeta{parts}
}

func (b buildMeta) compare(subj buildMeta) int {
	bLen, subjLen := len(b.parts), len(subj.parts)
	if bLen == 0 && subjLen > 0 {
		return greaterThan
	} else if bLen > 0 && subjLen == 0 {
		return lessThan
	}

	minLen := min(bLen, subjLen)
	for i := 0; i < minLen; i++ {
		if result := b.parts[i].compare(subj.parts[i]); result != equalTo {
			return result
		}
	}

	if bLen > subjLen {
		return greaterThan
	} else if subjLen > bLen {
		return lessThan
	}
	return equalTo
}

type buildMetaPart struct {
	numeric    bool
	numericVal uint64

	alphanumVal string
}

func newBuildMetaPart(part string) buildMetaPart {
	if isFullyNumeric(part) {
		val, _ := strconv.ParseUint(part, 10, 64)
		return buildMetaPart{numeric: true, numericVal: val}
	}
	return buildMetaPart{alphanumVal: part}
}

func (b buildMetaPart) compare(subj buildMetaPart) int {
	switch {
	case b.numeric && subj.numeric:
		switch {
		case b.numericVal < subj.numericVal:
			return lessThan
		case b.numericVal > subj.numericVal:
			return greaterThan
		default:
			return equalTo
		}
	case b.numeric:
		return lessThan
	case subj.numeric:
		return greaterThan
	default:
		return strings.Compare(b.alphanumVal, subj.alphanumVal)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isFullyNumeric(a string) bool {
	const numbers = "0123456789"
	for _, c := range a {
		if !strings.ContainsRune(numbers, c) {
			return false
		}
	}
	return true
}
