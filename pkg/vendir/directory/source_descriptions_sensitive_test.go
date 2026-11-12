// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package directory_test

import (
	"io"
	"strings"
	"testing"

	conf "carvel.dev/vendir/pkg/vendir/config"
	"carvel.dev/vendir/pkg/vendir/fetch/git"
	"carvel.dev/vendir/pkg/vendir/fetch/hg"
)

func TestSourceDescriptionsOmitCredentials(t *testing.T) {
	raw := "https://user:password-canary@example.com/source?token=query-canary"
	descriptions := []string{
		git.NewSync(conf.DirectoryContentsGit{URL: raw, Ref: "main"}, io.Discard, nil, nil).Desc(),
		hg.NewSync(conf.DirectoryContentsHg{URL: raw, Ref: "main"}, io.Discard, nil, nil).Desc(),
	}
	for _, desc := range descriptions {
		if strings.Contains(desc, "canary") || !strings.Contains(desc, "example.com/source") {
			t.Fatal("unsafe source description")
		}
	}
}
