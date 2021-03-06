// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config

type LockDirectory struct {
	Path     string                  `json:"path"`
	Contents []LockDirectoryContents `json:"contents"`
}

type LockDirectoryContents struct {
	Path string `json:"path"`

	Git                      *LockDirectoryContentsGit           `json:"git,omitempty"`
	HTTP                     *LockDirectoryContentsHTTP          `json:"http,omitempty"`
	Image                    *LockDirectoryContentsImage         `json:"image,omitempty"`
	ImgpkgBundle             *LockDirectoryContentsImgpkgBundle  `json:"imgpkgBundle,omitempty"`
	GithubRelease            *LockDirectoryContentsGithubRelease `json:"githubRelease,omitempty"`
	HelmChart                *LockDirectoryContentsHelmChart     `json:"helmChart,omitempty"`
	Manual                   *LockDirectoryContentsManual        `json:"manual,omitempty"`
	Directory                *LockDirectoryContentsDirectory     `json:"directory,omitempty"`
	Inline                   *LockDirectoryContentsInline        `json:"inline,omitempty"`
	TanzuNetworkProductFiles *LockDirectoryContentsTanzuNetwork  `json:"tanzuNetworkProductFiles,omitempty"`
}

type LockDirectoryContentsGit struct {
	SHA         string   `json:"sha"`
	Tags        []string `json:"tags,omitempty"`
	CommitTitle string   `json:"commitTitle"`
}

type LockDirectoryContentsHTTP struct{}

type LockDirectoryContentsImage struct {
	URL string `json:"url"`
}

type LockDirectoryContentsImgpkgBundle struct {
	Image string `json:"image"`
}

type LockDirectoryContentsGithubRelease struct {
	URL string `json:"url"`
}

type LockDirectoryContentsHelmChart struct {
	Version    string `json:"version"`
	AppVersion string `json:"appVersion"`
}

type LockDirectoryContentsTanzuNetwork struct {
	Slug      string                                  `json:"slug"`
	Version   string                                  `json:"version"`
	ReleaseID int                                     `json:"releaseID"`
	Files     []LockDirectoryContentsTanzuNetworkFile `json:"files"`
}

type LockDirectoryContentsTanzuNetworkFile struct {
	Name      string `json:"name"`
	ID        int    `json:"id"`
	SHA256Sum string `json:"sha246Sum"`
}

type LockDirectoryContentsManual struct{}

type LockDirectoryContentsDirectory struct{}

type LockDirectoryContentsInline struct{}
