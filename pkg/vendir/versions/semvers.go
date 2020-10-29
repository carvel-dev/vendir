package versions

import (
	"fmt"
	"sort"

	semver "github.com/hashicorp/go-version"
	// "github.com/blang/semver/v4"
)

type Semvers struct {
	versions []*semver.Version
}

func NewSemvers(versions []string) Semvers {
	var parsedVersions []*semver.Version

	for _, vStr := range versions {
		ver, err := semver.NewVersion(vStr)
		if err == nil {
			// Ignore non-parseable versions
			parsedVersions = append(parsedVersions, ver)
		}
	}

	return Semvers{parsedVersions}
}

func (v Semvers) Sorted() Semvers {
	var versions []*semver.Version

	for _, ver := range v.versions {
		versions = append(versions, ver)
	}

	sort.SliceStable(versions, func(i, j int) bool {
		return versions[i].LessThan(versions[j])
	})

	return Semvers{versions}
}

func (v Semvers) Filtered(constraintList string) (Semvers, error) {
	constraints, err := semver.NewConstraint(constraintList)
	if err != nil {
		return Semvers{}, fmt.Errorf("Parsing version constraint '%s': %s", constraintList, err)
	}

	var matchingVersions []*semver.Version

	for _, ver := range v.versions {
		if constraints.Check(ver) {
			matchingVersions = append(matchingVersions, ver)
		}
	}

	return Semvers{matchingVersions}, nil
}

func (v Semvers) Highest() (string, bool) {
	v = v.Sorted()

	if len(v.versions) == 0 {
		return "", false
	}

	return v.versions[len(v.versions)-1].Original(), true
}

func (v Semvers) All() []string {
	var verStrs []string
	for _, ver := range v.versions {
		verStrs = append(verStrs, ver.Original())
	}
	return verStrs
}
