package directory

import (
	"fmt"
	"strings"
)

const (
	EntireDirPath = "."
)

var (
	DefaultLegalPaths = []string{
		"{LICENSE,LICENCE,License,Licence}{,.md,.txt,.rst}",
		"{COPYRIGHT,Copyright}{,.md,.txt,.rst}",
		"{NOTICE,Notice}{,.md,.txt,.rst}",
	}

	disallowedPaths = []string{"/", EntireDirPath, "..", ""}
)

type Config struct {
	Path     string
	Contents []ConfigContents
}

type ConfigContents struct {
	Path string

	Git           *ConfigContentsGit
	GithubRelease *ConfigContentsGithubRelease
	Manual        *ConfigContentsManual
	Directory     *ConfigContentsDirectory

	IncludePaths []string `json:"includePaths"`
	ExcludePaths []string `json:"excludePaths"`

	// By default LICENSE/LICENCE/NOTICE/COPYRIGHT files are kept
	LegalPaths []string `json:"legalPaths"`
}

type ConfigContentsGit struct {
	URL string
	Ref string
}

type ConfigContentsGithubRelease struct {
	Slug string // e.g. organization/repository
	Tag  string

	Checksums                     map[string]string
	DisableAutoChecksumValidation bool `json:"disableAutoChecksumValidation"`

	UnpackArchive *ConfigContentsUnpackArchive `json:"unpackArchive"`
}

type ConfigContentsManual struct{}

type ConfigContentsDirectory struct {
	Path string
}

type ConfigContentsUnpackArchive struct {
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
			return fmt.Errorf("Expected only one directory contents if path is set to '%s'", EntireDirPath)
		}
	}

	return nil
}

func (c ConfigContents) Validate() error {
	// entire dir path is allowed for contents
	if c.Path != EntireDirPath {
		err := isDisallowedPath(c.Path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c ConfigContents) IsEntireDir() bool {
	return c.Path == EntireDirPath
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
