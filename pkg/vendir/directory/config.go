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
	HTTP          *ConfigContentsHTTP          `json:"http,omitempty"`
	Image         *ConfigContentsImage         `json:"image,omitempty"`
	GithubRelease *ConfigContentsGithubRelease `json:"githubRelease,omitempty"`
	HelmChart     *ConfigContentsHelmChart     `json:"helmChart,omitempty"`
	Manual        *ConfigContentsManual        `json:"manual,omitempty"`
	Directory     *ConfigContentsDirectory     `json:"directory,omitempty"`

	IncludePaths []string `json:"includePaths,omitempty"`
	ExcludePaths []string `json:"excludePaths,omitempty"`

	// By default LICENSE/LICENCE/NOTICE/COPYRIGHT files are kept
	LegalPaths []string `json:"legalPaths,omitempty"`
}

type ConfigContentsGit struct {
	URL string `json:"url,omitempty"`
	Ref string `json:"ref,omitempty"`
	// Secret may include one or more keys: ssh-privatekey, ssh-knownhosts
	// +optional
	SecretRef *ConfigContentsLocalRef `json:"secretRef,omitempty"`
	// +optional
	LFSSkipSmudge bool `json:"lfsSkipSmudge,omitempty"`
}

type ConfigContentsHTTP struct {
	// URL can point to one of following formats: text, tgz, zip
	URL string `json:"url,omitempty"`
	// +optional
	SHA256 string `json:"sha256,omitempty"`
	// Secret may include one or more keys: username, password
	// +optional
	SecretRef *ConfigContentsLocalRef `json:"secretRef,omitempty"`
}

type ConfigContentsImage struct {
	// Example: username/app1-config:v0.1.0
	URL string `json:"url,omitempty"`
	// Secret may include one or more keys: username, password, token.
	// By default anonymous access is used for authentication.
	// TODO support docker config formated secret
	// +optional
	SecretRef *ConfigContentsLocalRef `json:"secretRef,omitempty"`
}

type ConfigContentsGithubRelease struct {
	Slug   string `json:"slug"` // e.g. organization/repository
	Tag    string `json:"tag"`
	Latest bool   `json:"latest,omitempty"`
	URL    string `json:"url,omitempty"`

	Checksums                     map[string]string `json:"checksums,omitempty"`
	DisableAutoChecksumValidation bool              `json:"disableAutoChecksumValidation,omitempty"`

	UnpackArchive *ConfigContentsUnpackArchive `json:"unpackArchive,omitempty"`
}

type ConfigContentsHelmChart struct {
	// Example: stable/redis
	Name string `json:"name,omitempty"`
	// +optional
	Version    string                       `json:"version,omitempty"`
	Repository *ConfigContentsHelmChartRepo `json:"repository,omitempty"`
}

type ConfigContentsHelmChartRepo struct {
	URL string `json:"url,omitempty"`
	// +optional
	SecretRef *ConfigContentsLocalRef `json:"secretRef,omitempty"`
}

type ConfigContentsManual struct{}

type ConfigContentsDirectory struct {
	Path string `json:"path"`
}

type ConfigContentsUnpackArchive struct {
	Path string `json:"path"`
}

type ConfigContentsLocalRef struct {
	Name string `json:"name,omitempty"`
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
	if c.HTTP != nil {
		srcTypes = append(srcTypes, "http")
	}
	if c.Image != nil {
		srcTypes = append(srcTypes, "image")
	}
	if c.GithubRelease != nil {
		srcTypes = append(srcTypes, "githubRelease")
	}
	if c.HelmChart != nil {
		srcTypes = append(srcTypes, "helmChart")
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
	case c.HTTP != nil:
		return c.HTTP.Lock(lockConfig.HTTP)
	case c.Image != nil:
		return c.Image.Lock(lockConfig.Image)
	case c.GithubRelease != nil:
		return c.GithubRelease.Lock(lockConfig.GithubRelease)
	case c.HelmChart != nil:
		return c.HelmChart.Lock(lockConfig.HelmChart)
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

func (c *ConfigContentsHTTP) Lock(lockConfig *LockConfigContentsHTTP) error {
	if lockConfig == nil {
		return fmt.Errorf("Expected HTTP lock configuration to be non-empty")
	}
	return nil
}

func (c *ConfigContentsImage) Lock(lockConfig *LockConfigContentsImage) error {
	if lockConfig == nil {
		return fmt.Errorf("Expected image lock configuration to be non-empty")
	}
	if len(lockConfig.URL) == 0 {
		return fmt.Errorf("Expected image URL to be non-empty")
	}
	c.URL = lockConfig.URL
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

func (c *ConfigContentsHelmChart) Lock(lockConfig *LockConfigContentsHelmChart) error {
	if lockConfig == nil {
		return fmt.Errorf("Expected helm chart lock configuration to be non-empty")
	}
	if len(lockConfig.Version) == 0 {
		return fmt.Errorf("Expected helm chart version to be non-empty")
	}
	c.Version = lockConfig.Version
	return nil
}
