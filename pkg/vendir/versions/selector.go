// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package versions

import (
	"fmt"
	"strings"

	"carvel.dev/vendir/pkg/vendir/versions/v1alpha1"
)

type ConstraintCallback struct {
	Constraint func(string) bool
	Name       string
}

func HighestConstrainedVersion(versions []string, config v1alpha1.VersionSelection) (string, error) {
	return HighestConstrainedVersionWithAdditionalConstraints(versions, config, []ConstraintCallback{})
}

func HighestConstrainedVersionWithAdditionalConstraints(versions []string, config v1alpha1.VersionSelection, additionalConstraints []ConstraintCallback) (string, error) {
	switch {
	case config.Semver != nil:
		var details []string

		matchedVers := NewRelaxedSemversNoErr(versions) // this is secretly just all the possible versions, we haven't matched anything yet.
		details = append(details, fmt.Sprintf("all=%d", matchedVers.Len()))

		matchedVers = matchedVers.FilterPrereleases(config.Semver.Prereleases)
		details = append(details, fmt.Sprintf("after-prereleases-filter=%d", matchedVers.Len()))

		for _, check := range additionalConstraints {
			matchedVers = matchedVers.Filter(check.Constraint)
			details = append(details, fmt.Sprintf("after-%s=%d", check.Name, matchedVers.Len()))
		}

		if len(config.Semver.Constraints) > 0 {
			var err error
			matchedVers, err = matchedVers.FilterConstraints(config.Semver.Constraints)
			if err != nil {
				return "", fmt.Errorf("Selecting versions: %s", err)
			}
			details = append(details, fmt.Sprintf("after-constraints-filter=%d", matchedVers.Len()))
		}

		highestVersion, found := matchedVers.Highest()
		if !found {
			errFmt := "Expected to find at least one version, but did not (details: %s)"
			return "", fmt.Errorf(errFmt, strings.Join(details, " -> "))
		}
		return highestVersion, nil

	default:
		return "", fmt.Errorf("Unsupported version selection type (currently supported: semver)")
	}
}
