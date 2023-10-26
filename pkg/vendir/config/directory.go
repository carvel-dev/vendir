// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"strings"

	ctlver "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions/v1alpha1"
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

type Directory struct {
	Path     string              `json:"path"`
	Contents []DirectoryContents `json:"contents,omitempty"`

	Permissions *os.FileMode `json:"permissions,omitempty"`
}

type DirectoryContents struct {
	Path string `json:"path"`
	Lazy bool   `json:"lazy,omitempty"`

	Git           *DirectoryContentsGit           `json:"git,omitempty"`
	Hg            *DirectoryContentsHg            `json:"hg,omitempty"`
	HTTP          *DirectoryContentsHTTP          `json:"http,omitempty"`
	Image         *DirectoryContentsImage         `json:"image,omitempty"`
	ImgpkgBundle  *DirectoryContentsImgpkgBundle  `json:"imgpkgBundle,omitempty"`
	GithubRelease *DirectoryContentsGithubRelease `json:"githubRelease,omitempty"`
	HelmChart     *DirectoryContentsHelmChart     `json:"helmChart,omitempty"`
	Manual        *DirectoryContentsManual        `json:"manual,omitempty"`
	Directory     *DirectoryContentsDirectory     `json:"directory,omitempty"`
	Inline        *DirectoryContentsInline        `json:"inline,omitempty"`

	IncludePaths []string `json:"includePaths,omitempty"`
	ExcludePaths []string `json:"excludePaths,omitempty"`
	IgnorePaths  []string `json:"ignorePaths,omitempty"`

	// By default LICENSE/LICENCE/NOTICE/COPYRIGHT files are kept
	LegalPaths *[]string `json:"legalPaths,omitempty"`

	NewRootPath string `json:"newRootPath,omitempty"`

	Permissions *os.FileMode `json:"permissions,omitempty"`
}

type DirectoryContentsGit struct {
	URL          string                            `json:"url,omitempty"`
	Ref          string                            `json:"ref,omitempty"`
	RefSelection *ctlver.VersionSelection          `json:"refSelection,omitempty"`
	Verification *DirectoryContentsGitVerification `json:"verification,omitempty"`
	// Secret may include one or more keys: ssh-privatekey, ssh-knownhosts
	// +optional
	SecretRef *DirectoryContentsLocalRef `json:"secretRef,omitempty"`
	// +optional
	LFSSkipSmudge          bool `json:"lfsSkipSmudge,omitempty"`
	DangerousSkipTLSVerify bool `json:"dangerousSkipTLSVerify,omitempty"`
	SkipInitSubmodules     bool `json:"skipInitSubmodules,omitempty"`
	Depth                  int  `json:"depth,omitempty"`
}

type DirectoryContentsGitVerification struct {
	PublicKeysSecretRef *DirectoryContentsLocalRef `json:"publicKeysSecretRef,omitempty"`
}

type DirectoryContentsHg struct {
	URL    string `json:"url,omitempty"`
	Ref    string `json:"ref,omitempty"`
	Evolve bool   `json:"evolve,omitempty"`
	// Secret may include one or more keys: ssh-privatekey, ssh-knownhosts
	// +optional
	SecretRef *DirectoryContentsLocalRef `json:"secretRef,omitempty"`
}

type DirectoryContentsHgVerification struct {
	PublicKeysSecretRef *DirectoryContentsLocalRef `json:"publicKeysSecretRef,omitempty"`
}

type DirectoryContentsHTTP struct {
	// URL can point to one of following formats: text, tgz, zip
	URL string `json:"url,omitempty"`
	// +optional
	SHA256 string `json:"sha256,omitempty"`
	// Secret may include one or more keys: username, password
	// +optional
	SecretRef *DirectoryContentsLocalRef `json:"secretRef,omitempty"`
	// +optional
	DisableUnpack bool `json:"disableUnpack,omitempty"`
}

type DirectoryContentsImage struct {
	// Example: username/app1-config:v0.1.0
	URL string `json:"url,omitempty"`

	TagSelection   *ctlver.VersionSelection `json:"tagSelection,omitempty"`
	preresolvedTag string                   `json:"-"`

	// Secret may include one or more keys: username, password, token.
	// By default anonymous access is used for authentication.
	// TODO support docker config formated secret
	// +optional
	SecretRef *DirectoryContentsLocalRef `json:"secretRef,omitempty"`

	DangerousSkipTLSVerify bool `json:"dangerousSkipTLSVerify,omitempty"`
}

func (c DirectoryContentsImage) PreresolvedTag() string { return c.preresolvedTag }

type DirectoryContentsImgpkgBundle struct {
	// Example: username/app1-config:v0.1.0
	Image string `json:"image,omitempty"`

	TagSelection   *ctlver.VersionSelection `json:"tagSelection,omitempty"`
	preresolvedTag string                   `json:"-"`

	// Secret may include one or more keys: username, password, token.
	// By default anonymous access is used for authentication.
	// TODO support docker config formated secret
	// +optional
	SecretRef *DirectoryContentsLocalRef `json:"secretRef,omitempty"`

	DangerousSkipTLSVerify bool `json:"dangerousSkipTLSVerify,omitempty"`
	Recursive              bool `json:"recursive,omitempty"`
}

func (c DirectoryContentsImgpkgBundle) PreresolvedTag() string { return c.preresolvedTag }

type DirectoryContentsGithubRelease struct {
	Slug         string                   `json:"slug"` // e.g. organization/repository
	Tag          string                   `json:"tag"`
	TagSelection *ctlver.VersionSelection `json:"tagSelection,omitempty"`
	Latest       bool                     `json:"latest,omitempty"`
	URL          string                   `json:"url,omitempty"`

	Checksums                     map[string]string `json:"checksums,omitempty"`
	DisableAutoChecksumValidation bool              `json:"disableAutoChecksumValidation,omitempty"`

	AssetNames    []string                        `json:"assetNames,omitempty"`
	UnpackArchive *DirectoryContentsUnpackArchive `json:"unpackArchive,omitempty"`

	// Secret may include one key: token
	// +optional
	SecretRef *DirectoryContentsLocalRef `json:"secretRef,omitempty"`

	// +optional
	HTTP *DirectoryContentsHTTP `json:"http,omitempty"`
}

type DirectoryContentsHelmChart struct {
	// Example: stable/redis
	Name string `json:"name,omitempty"`
	// +optional
	Version    string                          `json:"version,omitempty"`
	Repository *DirectoryContentsHelmChartRepo `json:"repository,omitempty"`

	// +optional
	HelmVersion string `json:"helmVersion,omitempty"`
}

type DirectoryContentsHelmChartRepo struct {
	URL string `json:"url,omitempty"`
	// +optional
	SecretRef *DirectoryContentsLocalRef `json:"secretRef,omitempty"`
}

type DirectoryContentsManual struct{}

type DirectoryContentsDirectory struct {
	Path string `json:"path"`
}

type DirectoryContentsInline struct {
	Paths     map[string]string               `json:"paths,omitempty"`
	PathsFrom []DirectoryContentsInlineSource `json:"pathsFrom,omitempty"`
}

type DirectoryContentsInlineSource struct {
	SecretRef    *DirectoryContentsInlineSourceRef `json:"secretRef,omitempty"`
	ConfigMapRef *DirectoryContentsInlineSourceRef `json:"configMapRef,omitempty"`
}

type DirectoryContentsInlineSourceRef struct {
	DirectoryPath             string `json:"directoryPath,omitempty"`
	DirectoryContentsLocalRef `json:",inline"`
}

type DirectoryContentsUnpackArchive struct {
	Path string `json:"path"`
}

type DirectoryContentsLocalRef struct {
	Name string `json:"name,omitempty"`
}

func (c Directory) Validate() error {
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

func (c DirectoryContents) Validate() error {
	var srcTypes []string

	if c.Git != nil {
		srcTypes = append(srcTypes, "git")
	}
	if c.Hg != nil {
		srcTypes = append(srcTypes, "hg")
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
	if c.Inline != nil {
		srcTypes = append(srcTypes, "inline")
	}
	if c.ImgpkgBundle != nil {
		srcTypes = append(srcTypes, "imgpkgBundle")
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

func (c DirectoryContents) IsEntireDir() bool {
	return c.Path == EntireDirPath
}

func (c DirectoryContents) LegalPathsWithDefaults() []string {
	if c.LegalPaths == nil {
		return append([]string{}, DefaultLegalPaths...)
	}
	return *c.LegalPaths
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

func (c DirectoryContents) Lock(lockConfig LockDirectoryContents) error {
	switch {
	case c.Git != nil:
		return c.Git.Lock(lockConfig.Git)
	case c.Hg != nil:
		return c.Hg.Lock(lockConfig.Hg)
	case c.HTTP != nil:
		return c.HTTP.Lock(lockConfig.HTTP)
	case c.Image != nil:
		return c.Image.Lock(lockConfig.Image)
	case c.ImgpkgBundle != nil:
		return c.ImgpkgBundle.Lock(lockConfig.ImgpkgBundle)
	case c.GithubRelease != nil:
		return c.GithubRelease.Lock(lockConfig.GithubRelease)
	case c.HelmChart != nil:
		return c.HelmChart.Lock(lockConfig.HelmChart)
	case c.Directory != nil:
		return nil // nothing to lock
	case c.Manual != nil:
		return nil // nothing to lock
	case c.Inline != nil:
		return nil // nothing to lock
	default:
		panic("Unknown contents type")
	}
}

func (c *DirectoryContentsGit) Lock(lockConfig *LockDirectoryContentsGit) error {
	if lockConfig == nil {
		return fmt.Errorf("Expected git lock configuration to be non-empty")
	}
	if len(lockConfig.SHA) == 0 {
		return fmt.Errorf("Expected git SHA to be non-empty")
	}
	c.Ref = lockConfig.SHA
	return nil
}

func (c *DirectoryContentsHg) Lock(lockConfig *LockDirectoryContentsHg) error {
	if lockConfig == nil {
		return fmt.Errorf("Expected hg lock configuration to be non-empty")
	}
	if len(lockConfig.SHA) == 0 {
		return fmt.Errorf("Expected hg SHA to be non-empty")
	}
	c.Ref = lockConfig.SHA
	return nil
}

func (c *DirectoryContentsHTTP) Lock(lockConfig *LockDirectoryContentsHTTP) error {
	if lockConfig == nil {
		return fmt.Errorf("Expected HTTP lock configuration to be non-empty")
	}
	return nil
}

func (c *DirectoryContentsImage) Lock(lockConfig *LockDirectoryContentsImage) error {
	if lockConfig == nil {
		return fmt.Errorf("Expected image lock configuration to be non-empty")
	}
	if len(lockConfig.URL) == 0 {
		return fmt.Errorf("Expected image URL to be non-empty")
	}
	c.URL = lockConfig.URL
	c.TagSelection = nil // URL is fully resolved already
	c.preresolvedTag = lockConfig.Tag
	return nil
}

func (c *DirectoryContentsImgpkgBundle) Lock(lockConfig *LockDirectoryContentsImgpkgBundle) error {
	if lockConfig == nil {
		return fmt.Errorf("Expected image lock configuration to be non-empty")
	}
	if len(lockConfig.Image) == 0 {
		return fmt.Errorf("Expected imgpkg bundle Image to be non-empty")
	}
	c.Image = lockConfig.Image
	c.TagSelection = nil // URL is fully resolved already
	c.preresolvedTag = lockConfig.Tag
	return nil
}

func (c *DirectoryContentsGithubRelease) Lock(lockConfig *LockDirectoryContentsGithubRelease) error {
	if lockConfig == nil {
		return fmt.Errorf("Expected github release lock configuration to be non-empty")
	}
	if len(lockConfig.URL) == 0 {
		return fmt.Errorf("Expected github release URL to be non-empty")
	}
	c.URL = lockConfig.URL
	c.Tag = lockConfig.Tag
	return nil
}

func (c *DirectoryContentsHelmChart) Lock(lockConfig *LockDirectoryContentsHelmChart) error {
	if lockConfig == nil {
		return fmt.Errorf("Expected helm chart lock configuration to be non-empty")
	}
	if len(lockConfig.Version) == 0 {
		return fmt.Errorf("Expected helm chart version to be non-empty")
	}
	c.Version = lockConfig.Version
	return nil
}
