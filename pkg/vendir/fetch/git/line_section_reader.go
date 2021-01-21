// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"fmt"
	"strings"
)

type lineSectionReader struct {
	StartLine   string
	EndLine     string
	Description string
}

func (r lineSectionReader) Read(contents string, required bool) (string, string, error) {
	var outsideLines, sectionLines []string
	var opened, closed bool

	for _, line := range strings.Split(contents, "\n") {
		switch {
		case line == r.StartLine:
			if opened {
				return "", "", fmt.Errorf("Expected section to be closed before opening")
			}
			opened = true
			sectionLines = append(sectionLines, line)

		case line == r.EndLine:
			if !opened {
				return "", "", fmt.Errorf("Expected section to be opened before closing")
			}
			closed = true
			sectionLines = append(sectionLines, line)

		case opened && !closed:
			sectionLines = append(sectionLines, line)

		default:
			outsideLines = append(outsideLines, line)
		}
	}

	if opened && !closed {
		return "", "", fmt.Errorf("Expected section to be closed before ending")
	}
	if required && !opened {
		return "", "", fmt.Errorf("Expected to find section '%s', but did not", r.Description)
	}

	return strings.Join(outsideLines, "\n"), strings.Join(sectionLines, "\n"), nil
}
