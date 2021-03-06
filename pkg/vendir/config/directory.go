// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
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
}

type DirectoryContents struct {
	Path string `json:"path"`

	Git           *DirectoryContentsGit           `json:"git,omitempty"`
	HTTP          *DirectoryContentsHTTP          `json:"http,omitempty"`
	Image         *DirectoryContentsImage         `json:"image,omitempty"`
	ImgpkgBundle  *DirectoryContentsImgpkgBundle  `json:"imgpkgBundle,omitempty"`
	GithubRelease *DirectoryContentsGithubRelease `json:"githubRelease,omitempty"`
	HelmChart     *DirectoryContentsHelmChart     `json:"helmChart,omitempty"`
	Manual        *DirectoryContentsManual        `json:"manual,omitempty"`
	Directory     *DirectoryContentsDirectory     `json:"directory,omitempty"`
	Inline        *DirectoryContentsInline        `json:"inline,omitempty"`
	TanzuNetwork  *DirectoryContentsTanzuNetwork  `json:"tanzuNetwork,omitempty"`

	IncludePaths []string `json:"includePaths,omitempty"`
	ExcludePaths []string `json:"excludePaths,omitempty"`

	// By default LICENSE/LICENCE/NOTICE/COPYRIGHT files are kept
	LegalPaths []string `json:"legalPaths,omitempty"`

	NewRootPath string `json:"newRootPath,omitempty"`
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
	LFSSkipSmudge bool `json:"lfsSkipSmudge,omitempty"`
}

type DirectoryContentsGitVerification struct {
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
}

type DirectoryContentsImage struct {
	// Example: username/app1-config:v0.1.0
	URL string `json:"url,omitempty"`
	// Secret may include one or more keys: username, password, token.
	// By default anonymous access is used for authentication.
	// TODO support docker config formated secret
	// +optional
	SecretRef *DirectoryContentsLocalRef `json:"secretRef,omitempty"`
}

type DirectoryContentsImgpkgBundle struct {
	// Example: username/app1-config:v0.1.0
	Image string `json:"image,omitempty"`
	// Secret may include one or more keys: username, password, token.
	// By default anonymous access is used for authentication.
	// TODO support docker config formated secret
	// +optional
	SecretRef *DirectoryContentsLocalRef `json:"secretRef,omitempty"`
}

type DirectoryContentsGithubRelease struct {
	Slug   string `json:"slug"` // e.g. organization/repository
	Tag    string `json:"tag"`
	Latest bool   `json:"latest,omitempty"`
	URL    string `json:"url,omitempty"`

	Checksums                     map[string]string `json:"checksums,omitempty"`
	DisableAutoChecksumValidation bool              `json:"disableAutoChecksumValidation,omitempty"`

	AssetNames    []string                        `json:"assetNames,omitempty"`
	UnpackArchive *DirectoryContentsUnpackArchive `json:"unpackArchive,omitempty"`

	// Secret may include one key: token
	// +optional
	SecretRef *DirectoryContentsLocalRef `json:"secretRef,omitempty"`
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

type DirectoryContentsTanzuNetwork struct {
	Slug string `json:"slug"`

	// *optional
	Version string `json:"version"`

	// *optional
	ReleaseID *int `json:"releaseID"`

	Files []DirectoryContentsTanzuNetworkFile `json:"files"`
}

type DirectoryContentsTanzuNetworkFile struct {
	Name string `json:"name"`

	// +optional
	ID *int `json:"id,omitempty"`
	// +optional
	SHA256Sum *string `json:"sha246Sum,omitempty"`
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
	if c.TanzuNetwork != nil {
		srcTypes = append(srcTypes, "tanzuNetwork")
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

func (c DirectoryContents) Lock(lockConfig LockDirectoryContents) error {
	switch {
	case c.Git != nil:
		return c.Git.Lock(lockConfig.Git)
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
	case c.TanzuNetwork != nil:
		return c.TanzuNetwork.Lock(lockConfig.TanzuNetworkProductFiles)
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

func (c *DirectoryContentsTanzuNetwork) Lock(lockConfig *LockDirectoryContentsTanzuNetwork) error {
	if lockConfig == nil {
		return fmt.Errorf("Expected Tanzu Network product file configuration to be non-empty")
	}

	c.Slug = lockConfig.Slug
	c.ReleaseID = &lockConfig.ReleaseID

	c.Files = make([]DirectoryContentsTanzuNetworkFile, len(lockConfig.Files))
	for i, lock := range lockConfig.Files {
		err := c.Files[i].Lock(&lock)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *DirectoryContentsTanzuNetworkFile) Lock(lockConfig *LockDirectoryContentsTanzuNetworkFile) error {
	if lockConfig.Name == "" {
		return fmt.Errorf("Expected Tanzu Network product file name to not be non-empty")
	}

	if lockConfig.SHA256Sum == "" {
		return fmt.Errorf("Expected Tanzu Network product file with name %q to have sha256sum not be non-empty", lockConfig.Name)
	}

	if lockConfig.ID == 0 {
		// Are there valid file id's with value 0? I'm not sure.
		return fmt.Errorf("Expected Tanzu Network product file with name %q to have sha256sum not be non-empty (non-zero)", lockConfig.Name)
	}

	c.Name = lockConfig.Name
	c.SHA256Sum = &lockConfig.SHA256Sum
	c.ID = &lockConfig.ID

	return nil
}
