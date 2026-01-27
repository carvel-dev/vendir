// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package versions

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"carvel.dev/vendir/pkg/vendir/versions/v1alpha1"
	semver "github.com/carvel-dev/semver/v4"
)

type Semvers struct {
	versions []SemverWrap
}

type SemverWrap struct {
	semver.Version
	Original string
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

// chunkRegexp helps split metadata into numeric and non-numeric parts for natural sorting.
var chunkRegexp = regexp.MustCompile(`(\d+|\D+)`)

func (v Semvers) Sorted() Semvers {
	var versions []SemverWrap
	for _, ver := range v.versions {
		versions = append(versions, ver)
	}

	sort.SliceStable(versions, func(i, j int) bool {
		vI := versions[i].Version
		vJ := versions[j].Version

		// 1. Compare Major, Minor, Patch (Numeric)
		if vI.Major != vJ.Major {
			return vI.Major < vJ.Major
		}
		if vI.Minor != vJ.Minor {
			return vI.Minor < vJ.Minor
		}
		if vI.Patch != vJ.Patch {
			return vI.Patch < vJ.Patch
		}

		// 2. Standard Semver Pre-release Comparison (Spec 2.0.0)
		// A version with a pre-release is LOWER than one without.
		preLenI := len(vI.Pre)
		preLenJ := len(vJ.Pre)
		if preLenI > 0 && preLenJ == 0 {
			return true
		}
		if preLenI == 0 && preLenJ > 0 {
			return false
		}

		// If both have standard pre-releases, compare them using library logic
		if preLenI > 0 && preLenJ > 0 {
			// This logic is adapted from the semver library's Compare function
			for k := 0; k < preLenI && k < preLenJ; k++ {
				pI, pJ := vI.Pre[k], vJ.Pre[k]
				if pI.IsNum != pJ.IsNum {
					return pI.IsNum
				}
				if pI.IsNum {
					if pI.VersionNum != pJ.VersionNum {
						return pI.VersionNum < pJ.VersionNum
					}
				} else {
					if pI.VersionStr != pJ.VersionStr {
						return pI.VersionStr < pJ.VersionStr
					}
				}
			}
			if preLenI != preLenJ {
				return preLenI < preLenJ
			}
		}

		// 3. Relaxed Metadata Logic
		rawI := strings.ToLower(versions[i].Original)
		rawJ := strings.ToLower(versions[j].Original)

		isPre := func(s string) bool {
			return strings.Contains(s, "-rc") || strings.Contains(s, "-alpha") || strings.Contains(s, "-beta")
		}

		if rawI != rawJ {
			preI, preJ := isPre(rawI), isPre(rawJ)

			// Helper to get base version without RC/Alpha/Beta tags
			getBase := func(s string) string {
				s = strings.Split(s, "-rc")[0]
				s = strings.Split(s, "-alpha")[0]
				s = strings.Split(s, "-beta")[0]
				return s
			}

			baseI, baseJ := getBase(rawI), getBase(rawJ)

			// If they share the same metadata base, the one with the RC tag is LOWER.
			if baseI == baseJ && preI != preJ {
				return preI // If I is pre, it is "less" (true)
			}

			// 4. Natural Sort Fallback for everything else (9 vs 10, etc.)
			return naturalLess(rawI, rawJ)
		}

		return false
	})

	return Semvers{versions}
}

func naturalLess(s1, s2 string) bool {
	// If strings are identical, neither is "less"
	if s1 == s2 {
		return false
	}

	chunks1 := chunkRegexp.FindAllString(s1, -1)
	chunks2 := chunkRegexp.FindAllString(s2, -1)

	len1 := len(chunks1)
	len2 := len(chunks2)
	n := len1
	if len2 < n {
		n = len2
	}

	for i := 0; i < n; i++ {
		c1, c2 := chunks1[i], chunks2[i]
		if c1 == c2 {
			continue
		}

		n1, err1 := strconv.Atoi(c1)
		n2, err2 := strconv.Atoi(c2)

		// If both chunks are numeric, compare as integers
		if err1 == nil && err2 == nil {
			if n1 != n2 {
				return n1 < n2
			}
			continue
		}

		// Alphabetical comparison if one or both aren't numbers
		return c1 < c2
	}

	// If all chunks matched up to the length of the shorter string,
	// the shorter string is "less".
	return len1 < len2
}

func (v Semvers) FilterConstraints(constraintList string) (Semvers, error) {
	constraints, err := semver.ParseRange(constraintList)
	if err != nil {
		return Semvers{}, fmt.Errorf("Parsing version constraint '%s': %s", constraintList, err)
	}

	var matchingVersions []SemverWrap

	for _, ver := range v.versions {
		if constraints(ver.Version) {
			matchingVersions = append(matchingVersions, ver)
		}
	}

	return Semvers{matchingVersions}, nil
}

func (v Semvers) FilterPrereleases(prereleases *v1alpha1.VersionSelectionSemverPrereleases) Semvers {
	if prereleases == nil {
		// Exclude all prereleases
		var result []SemverWrap
		for _, ver := range v.versions {
			if len(ver.Version.Pre) == 0 {
				result = append(result, ver)
			}
		}
		return Semvers{result}
	}

	preIdentifiersAsMap := prereleases.IdentifiersAsMap()

	var result []SemverWrap
	for _, ver := range v.versions {
		if len(ver.Version.Pre) == 0 || v.shouldKeepPrerelease(ver.Version, preIdentifiersAsMap) {
			result = append(result, ver)
		}
	}
	return Semvers{result}
}

func (v Semvers) Filter(f func(string) bool) Semvers {
	var result []SemverWrap
	for _, ver := range v.versions {
		if f(ver.Original) {
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

func (v Semvers) Len() int { return len(v.versions) }
