package directory

import (
	"fmt"
	"strings"
)

const (
	entireDirPath = "."
)

var (
	DefaultLegalPaths = []string{
		"{LICENSE,LICENCE,License,Licence}{,.md,.txt,.rst}",
		"{COPYRIGHT,Copyright}{,.md,.txt,.rst}",
		"{NOTICE,Notice}{,.md,.txt,.rst}",
	}

	disallowedPaths = []string{"/", entireDirPath, "..", ""}
)

type Config struct {
	Path     string
	Contents []ConfigContents
}

type ConfigContents struct {
	Path string

	Git       *ConfigContentsGit
	Manual    *ConfigContentsManual
	Directory *ConfigContentsDirectory

	IncludePaths []string `json:"includePaths"`
	ExcludePaths []string `json:"excludePaths"`

	// By default LICENSE/LICENCE/NOTICE/COPYRIGHT files are kept
	LegalPaths []string `json:"legalPaths"`
}

type ConfigContentsGit struct {
	URL string
	Ref string
}

type ConfigContentsManual struct{}

type ConfigContentsDirectory struct {
	Path string
}

func (c Config) Validate() error {
	err := isDisallowedPath(c.Path)
	if err != nil {
		return err
	}

	{ // Check for consumption of entire directory
		var consumesEntireDir bool
		for _, con := range c.Contents {
			if con.IsEntireDir() {
				consumesEntireDir = true
			}
		}
		if consumesEntireDir && len(c.Contents) != 1 {
			return fmt.Errorf("Expected only one directory contents if path is set to '%s'", entireDirPath)
		}
	}

	return nil
}

func (c ConfigContents) Validate() error {
	// entire dir path is allowed for contents
	if c.Path != entireDirPath {
		err := isDisallowedPath(c.Path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c ConfigContents) IsEntireDir() bool {
	return c.Path == entireDirPath
}

func (c ConfigContents) LegalPathsWithDefaults() []string {
	if len(c.LegalPaths) == 0 {
		return append([]string{}, DefaultLegalPaths...)
	}
	return c.LegalPaths
}

func isDisallowedPath(path string) error {
	for _, p := range disallowedPaths {
		if path == p {
			return fmt.Errorf("Expected path to not be one of '%s'",
				strings.Join(disallowedPaths, "', '"))
		}
	}
	return nil
}
