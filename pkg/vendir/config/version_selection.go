package config

type VersionSelection struct {
	Semver *VersionSelectionSemver `json:"semVer,omitempty"`
}

type VersionSelectionSemver struct {
	Constraints string `json:"constraints,omitempty"`
}
