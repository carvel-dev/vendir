package directory

import (
	"fmt"
	"regexp"
	"strings"
)

type ReleaseNotesChecksums struct{}

func (ReleaseNotesChecksums) Find(files []string, body string) (map[string]string, error) {
	lines := strings.Split(body, "\n")
	results := map[string]string{}

	for _, file := range files {
		var found bool

		for _, line := range lines {
			// Matches sha256 checksums
			findChecksum := regexp.MustCompile("^\\s*([a-f0-9]{64})\\s+(\\/|\\.\\/)?" +
				regexp.QuoteMeta(file) + "\\s*$")

			matches := findChecksum.FindStringSubmatch(line)
			if len(matches) == 3 {
				results[file] = matches[1]
				found = true
				break
			}
		}

		if !found {
			return results, fmt.Errorf("Expected to find sha256 checksum for file '%s'", file)
		}
	}

	return results, nil
}
