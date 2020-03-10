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
	Path     string           `json:"path"`
	Contents []ConfigContents `json:"contents,omitempty"`
}

type ConfigContents struct {
	Path string `json:"path"`

	Git           *ConfigContentsGit           `json:"git,omitempty"`
	GithubRelease *ConfigContentsGithubRelease `json:"githubRelease,omitempty"`
	Manual        *ConfigContentsManual        `json:"manual,omitempty"`
	Directory     *ConfigContentsDirectory     `json:"directory,omitempty"`

	IncludePaths []string `json:"includePaths,omitempty"`
	ExcludePaths []string `json:"excludePaths,omitempty"`

	// By default LICENSE/LICENCE/NOTICE/COPYRIGHT files are kept
	LegalPaths []string `json:"legalPaths,omitempty"`
}

type ConfigContentsGit struct {
	URL string `json:"url"`
	Ref string `json:"ref"`
}

type ConfigContentsGithubRelease struct {
	Slug string `json:"slug"` // e.g. organization/repository
	Tag  string `json:"tag"`
	URL  string `json:"url,omitempty"`

	Checksums                     map[string]string `json:"checksums,omitempty"`
	DisableAutoChecksumValidation bool              `json:"disableAutoChecksumValidation,omitempty"`

	UnpackArchive *ConfigContentsUnpackArchive `json:"unpackArchive,omitempty"`
}

type ConfigContentsManual struct{}

type ConfigContentsDirectory struct {
	Path string `json:"path"`
}

type ConfigContentsUnpackArchive struct {
	Path string `json:"path"`
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

	for i, con := range c.Contents {
		err := con.Validate()
		if err != nil {
			return fmt.Errorf("Validating directory contents '%s' (%d): %s", con.Path, i, err)
		}
	}

	return nil
}

func (c ConfigContents) Validate() error {
	var srcTypes []string

	if c.Git != nil {
		srcTypes = append(srcTypes, "git")
	}
	if c.GithubRelease != nil {
		srcTypes = append(srcTypes, "githubRelease")
	}
	if c.Manual != nil {
		srcTypes = append(srcTypes, "manual")
	}
	if c.Directory != nil {
		srcTypes = append(srcTypes, "directory")
	}

	if len(srcTypes) == 0 {
		return fmt.Errorf("Expected directory contents type to be specified (one of git, manual, etc.)")
	}
	if len(srcTypes) > 1 {
		return fmt.Errorf("Expected exactly one directory contents type to be specified (multiple found: %s)", strings.Join(srcTypes, ", "))
	}

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

func (c ConfigContents) Lock(lockConfig LockConfigContents) error {
	switch {
	case c.Git != nil:
		return c.Git.Lock(lockConfig.Git)
	case c.GithubRelease != nil:
		return c.GithubRelease.Lock(lockConfig.GithubRelease)
	case c.Directory != nil:
		return nil // nothing to lock
	case c.Manual != nil:
		return nil // nothing to lock
	default:
		panic("Unknown contents type")
	}
}

func (c *ConfigContentsGit) Lock(lockConfig *LockConfigContentsGit) error {
	if lockConfig == nil {
		return fmt.Errorf("Expected git lock configuration to be non-empty")
	}
	if len(lockConfig.SHA) == 0 {
		return fmt.Errorf("Expected git SHA to be non-empty")
	}
	c.Ref = lockConfig.SHA
	return nil
}

func (c *ConfigContentsGithubRelease) Lock(lockConfig *LockConfigContentsGithubRelease) error {
	if lockConfig == nil {
		return fmt.Errorf("Expected github release lock configuration to be non-empty")
	}
	if len(lockConfig.URL) == 0 {
		return fmt.Errorf("Expected github release URL to be non-empty")
	}
	c.URL = lockConfig.URL
	return nil
}
