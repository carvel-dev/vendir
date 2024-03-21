// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package githubrelease

import (
	"fmt"
	"regexp"
	"strings"
)

type ReleaseNotesChecksums struct{}

func (ReleaseNotesChecksums) Find(assets []ReleaseAssetAPI, body string) (map[string]string, error) {
	lines := strings.Split(body, "\n")
	results := map[string]string{}

	for _, asset := range assets {
		var found bool

		for _, line := range lines {
			// Matches sha256 checksums
			findChecksum := regexp.MustCompile("^\\s*([a-f0-9]{64})\\s+(\\/|\\.\\/)?" +
				regexp.QuoteMeta(asset.Name) + "\\s*$")

			matches := findChecksum.FindStringSubmatch(line)
			if len(matches) == 3 {
				results[asset.Name] = matches[1]
				found = true
				break
			}
		}

		if !found {
			return results, fmt.Errorf("Expected to find sha256 checksum for file '%s'", asset.Name)
		}
	}

	return results, nil
}
