package config

type LockDirectory struct {
	Path     string                  `json:"path"`
	Contents []LockDirectoryContents `json:"contents"`
}

type LockDirectoryContents struct {
	Path string `json:"path"`

	Git           *LockDirectoryContentsGit           `json:"git,omitempty"`
	HTTP          *LockDirectoryContentsHTTP          `json:"http,omitempty"`
	Image         *LockDirectoryContentsImage         `json:"image,omitempty"`
	GithubRelease *LockDirectoryContentsGithubRelease `json:"githubRelease,omitempty"`
	HelmChart     *LockDirectoryContentsHelmChart     `json:"helmChart,omitempty"`
	Manual        *LockDirectoryContentsManual        `json:"manual,omitempty"`
	Directory     *LockDirectoryContentsDirectory     `json:"directory,omitempty"`
}

type LockDirectoryContentsGit struct {
	SHA         string `json:"sha"`
	CommitTitle string `json:"commitTitle"`
}

type LockDirectoryContentsHTTP struct{}

type LockDirectoryContentsImage struct {
	URL string `json:"url"`
}

type LockDirectoryContentsGithubRelease struct {
	URL string `json:"url"`
}

type LockDirectoryContentsHelmChart struct {
	Version    string `json:"version"`
	AppVersion string `json:"appVersion"`
}

type LockDirectoryContentsManual struct{}

type LockDirectoryContentsDirectory struct{}
