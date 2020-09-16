package githubrelease_test

import (
	"reflect"
	"testing"

	. "github.com/k14s/vendir/pkg/vendir/fetch/githubrelease"
)

func TestReleaseNotesChecksums(t *testing.T) {
	body := `
- Initial release
  - some content inside release.yml

+++
26bf09c42d72ae448af3d1ee9f6a933c87c4ec81d04d37b30e1b6a339f5983a1  release.yml
26bf09c42d72ae448af3d1ee9f6a933c87c4ec81d04d37b30e1b6a339f5983a2  /with-slash.yml
26bf09c42d72ae448af3d1ee9f6a933c87c4ec81d04d37b30e1b6a339f5983a3  ./with-period-slash.yml
26bf09c42d72ae448af3d1ee9f6a933c87c4ec81d04d37b30e1b6a339f5983a4  with-space-after-file.yml   
+++
`

	files := []string{"release.yml", "with-slash.yml",
		"with-period-slash.yml", "with-space-after-file.yml"}

	results, err := ReleaseNotesChecksums{}.Find(files, body)
	if err != nil {
		t.Fatalf("Expected to succeed, but was: %s", err)
	}

	expectedResults := map[string]string{
		"release.yml":               "26bf09c42d72ae448af3d1ee9f6a933c87c4ec81d04d37b30e1b6a339f5983a1",
		"with-slash.yml":            "26bf09c42d72ae448af3d1ee9f6a933c87c4ec81d04d37b30e1b6a339f5983a2",
		"with-period-slash.yml":     "26bf09c42d72ae448af3d1ee9f6a933c87c4ec81d04d37b30e1b6a339f5983a3",
		"with-space-after-file.yml": "26bf09c42d72ae448af3d1ee9f6a933c87c4ec81d04d37b30e1b6a339f5983a4",
	}

	if !reflect.DeepEqual(results, expectedResults) {
		t.Fatalf("Expected checksums to equal")
	}
}
