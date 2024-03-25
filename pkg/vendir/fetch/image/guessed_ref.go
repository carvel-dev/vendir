// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"regexp"
	"strings"
)

// GuessedRefParts represents an image ref that was guessed from a string.
// it may not be an accurate representation since imgpkg parses ref itself.
// (Did not want to use go-containerregistry to avoid taking extra dep)
type GuessedRefParts struct {
	Repo   string
	Tag    string
	Digest string
}

var (
	imageRefParts = regexp.MustCompile(`\A(.+?)(:[0-9a-zA-Z_\-\.]+)?(@[0-9a-z]+:[0-9a-z]+)?\z`)
)

func NewGuessedRefParts(ref string) GuessedRefParts {
	matches := imageRefParts.FindStringSubmatch(ref)
	if len(matches) != 4 {
		return GuessedRefParts{}
	}
	return GuessedRefParts{
		Repo:   matches[1],
		Tag:    strings.TrimPrefix(matches[2], ":"),
		Digest: strings.TrimPrefix(matches[3], "@"),
	}
}
