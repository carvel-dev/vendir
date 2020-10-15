package directory

import (
	"fmt"
	"sort"

	semver "github.com/hashicorp/go-version"
)

// yeah yeah it's double version
type SemverVersions struct {
	versions []*semver.Version
}

func NewSemverVersions(versions []string) SemverVersions {
	var parsedVersions []*semver.Version

	for _, vStr := range versions {
		ver, err := semver.NewVersion(vStr)
		if err == nil {
			// Ignore non-parseable versions
			parsedVersions = append(parsedVersions, ver)
		}
	}

	return SemverVersions{parsedVersions}
}

func (v SemverVersions) Sorted() SemverVersions {
	var versions []*semver.Version

	for _, ver := range v.versions {
		versions = append(versions, ver)
	}

	sort.SliceStable(versions, func(i, j int) bool {
		return versions[i].LessThan(versions[j])
	})

	return SemverVersions{versions}
}

func (v SemverVersions) Filtered(constraintList string) (SemverVersions, error) {
	constraints, err := semver.NewConstraint(constraintList)
	if err != nil {
		return SemverVersions{}, fmt.Errorf("Parsing version constraint '%s': %s", constraintList, err)
	}

	var matchingVersions []*semver.Version

	for _, ver := range v.versions {
		if constraints.Check(ver) {
			matchingVersions = append(matchingVersions, ver)
		}
	}

	return SemverVersions{matchingVersions}, nil
}

func (v SemverVersions) Highest() (string, bool) {
	v = v.Sorted()

	if len(v.versions) == 0 {
		return "", false
	}

	return v.versions[len(v.versions)-1].Original(), true
}

func (v SemverVersions) All() []string {
	var verStrs []string
	for _, ver := range v.versions {
		verStrs = append(verStrs, ver.Original())
	}
	return verStrs
}
