package directory

type LockConfig struct {
	Path     string               `json:"path"`
	Contents []LockConfigContents `json:"contents"`
}

type LockConfigContents struct {
	Path          string                           `json:"path"`
	Git           *LockConfigContentsGit           `json:"git,omitempty"`
	GithubRelease *LockConfigContentsGithubRelease `json:"githubRelease,omitempty"`
	Manual        *LockConfigContentsManual        `json:"manual,omitempty"`
	Directory     *LockConfigContentsDirectory     `json:"directory,omitempty"`
}

type LockConfigContentsGit struct {
	SHA         string `json:"sha"`
	CommitTitle string `json:"commitTitle"`
}

type LockConfigContentsGithubRelease struct {
	URL string `json:"url"`
}

type LockConfigContentsManual struct{}

type LockConfigContentsDirectory struct{}
