package versions

type VersionSelection struct {
	Semver *VersionSelectionSemver `json:"semver,omitempty"`
}

type VersionSelectionSemver struct {
	Constraints string                             `json:"constraints,omitempty"`
	Prereleases *VersionSelectionSemverPrereleases `json:"prereleases,omitempty"`
}

type VersionSelectionSemverPrereleases struct {
	Identifiers []string `json:"identifiers,omitempty"`
}

func (p VersionSelectionSemverPrereleases) IdentifiersAsMap() map[string]struct{} {
	result := map[string]struct{}{}
	for _, name := range p.Identifiers {
		result[name] = struct{}{}
	}
	return result
}
