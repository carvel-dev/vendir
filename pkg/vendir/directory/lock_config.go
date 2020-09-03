package directory

type LockConfig struct {
	Path     string               `json:"path"`
	Contents []LockConfigContents `json:"contents"`
}

type LockConfigContents struct {
	Path string `json:"path"`

	Git           *LockConfigContentsGit           `json:"git,omitempty"`
	HTTP          *LockConfigContentsHTTP          `json:"http,omitempty"`
	Image         *LockConfigContentsImage         `json:"image,omitempty"`
	GithubRelease *LockConfigContentsGithubRelease `json:"githubRelease,omitempty"`
	HelmChart     *LockConfigContentsHelmChart     `json:"helmChart,omitempty"`
	Manual        *LockConfigContentsManual        `json:"manual,omitempty"`
	Directory     *LockConfigContentsDirectory     `json:"directory,omitempty"`
}

type LockConfigContentsGit struct {
	SHA         string `json:"sha"`
	CommitTitle string `json:"commitTitle"`
}

type LockConfigContentsHTTP struct{}

type LockConfigContentsImage struct {
	URL string `json:"url"`
}

type LockConfigContentsGithubRelease struct {
	URL string `json:"url"`
}

type LockConfigContentsHelmChart struct {
	Version    string `json:"version"`
	AppVersion string `json:"appVersion"`
}

type LockConfigContentsManual struct{}

type LockConfigContentsDirectory struct{}
