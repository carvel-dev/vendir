package directory

type LockConfig struct {
	Path     string               `json:"path"`
	Contents []LockConfigContents `json:"contents"`
}

type LockConfigContents struct {
	Path   string                    `json:"path"`
	Git    *LockConfigContentsGit    `json:"git,omitempty"`
	Manual *LockConfigContentsManual `json:"manual,omitempty"`
}

type LockConfigContentsGit struct {
	SHA         string `json:"sha"`
	CommitTitle string `json:"commitTitle"`
}

type LockConfigContentsManual struct{}
