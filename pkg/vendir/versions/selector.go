// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package versions

import (
	"fmt"
	"strings"

	"github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions/v1alpha1"
)

func HighestConstrainedVersion(versions []string, config v1alpha1.VersionSelection) (string, error) {
	switch {
	case config.Semver != nil:
		var details []string

		matchedVers := NewRelaxedSemversNoErr(versions)
		details = append(details, fmt.Sprintf("all=%d", matchedVers.Len()))

		matchedVers = matchedVers.FilterPrereleases(config.Semver.Prereleases)
		details = append(details, fmt.Sprintf("after-prereleases-filter=%d", matchedVers.Len()))

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
