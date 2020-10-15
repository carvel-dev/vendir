package directory

import (
	"fmt"
	"sort"

	semver "github.com/hashicorp/go-version"
)

type Versions struct {
	versions []*semver.Version
}

func NewVersions(versions []string) Versions {
	var parsedVersions []*semver.Version

	for _, vStr := range versions {
		ver, err := semver.NewVersion(vStr)
		if err == nil {
			// Ignore non-parseable versions
			parsedVersions = append(parsedVersions, ver)
		}
	}

	return Versions{parsedVersions}
}

func (v Versions) Sorted() Versions {
	var versions []*semver.Version

	for _, ver := range v.versions {
		versions = append(versions, ver)
	}

	sort.SliceStable(versions, func(i, j int) bool {
		return versions[i].LessThan(versions[j])
	})

	return Versions{versions}
}

func (v Versions) Filtered(constraintList string) (Versions, error) {
	constraints, err := semver.NewConstraint(constraintList)
	if err != nil {
		return Versions{}, fmt.Errorf("Parsing version constraint '%s': %s", constraintList, err)
	}

	var matchingVersions []*semver.Version

	for _, ver := range v.versions {
		if constraints.Check(ver) {
			matchingVersions = append(matchingVersions, ver)
		}
	}

	return Versions{matchingVersions}, nil
}

func (v Versions) Highest() (string, bool) {
	v = v.Sorted()

	if len(v.versions) == 0 {
		return "", false
	}

	return v.versions[len(v.versions)-1].Original(), true
}

func (v Versions) All() []string {
	var verStrs []string
	for _, ver := range v.versions {
		verStrs = append(verStrs, ver.Original())
	}
	return verStrs
}
