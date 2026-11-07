// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package githubrelease

import (
	"strings"
	"testing"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
)

// A slug is taken from vendir.yml as written, and nothing validates its shape
// before it is split, so a missing organization used to be an index panic.
func TestFetchTagSelectionRejectsMalformedSlug(t *testing.T) {
	for _, slug := range []string{"norepo", "", "/repo", "owner/"} {
		d := Sync{opts: ctlconf.DirectoryContentsGithubRelease{Slug: slug}}

		_, err := d.fetchTagSelection()
		if err == nil {
			t.Errorf("slug %q: expected an error, got none", slug)
			continue
		}
		if !strings.Contains(err.Error(), "organization/repository") {
			t.Errorf("slug %q: expected a format error, got %v", slug, err)
		}
	}
}
