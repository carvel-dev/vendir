package directory

var (
	DefaultLegalPaths = []string{
		"{LICENSE,LICENCE,License,Licence}{,.md,.txt,.rst}",
		"{COPYRIGHT,Copyright}{,.md,.txt,.rst}",
		"{NOTICE,Notice}{,.md,.txt,.rst}",
	}
)

type Config struct {
	Path     string
	Contents []ConfigContents
}

type ConfigContents struct {
	Path string

	Git       *ConfigContentsGit
	Manual    *ConfigContentsManual
	Directory *ConfigContentsDirectory

	IncludePaths []string `json:"includePaths"`
	ExcludePaths []string `json:"excludePaths"`

	// By default LICENSE/LICENCE/NOTICE/COPYRIGHT files are kept
	LegalPaths []string `json:"legalPaths"`
}

type ConfigContentsGit struct {
	URL string
	Ref string
}

type ConfigContentsManual struct{}

type ConfigContentsDirectory struct {
	Path string
}

func (c ConfigContents) LegalPathsWithDefaults() []string {
	if len(c.LegalPaths) == 0 {
		return append([]string{}, DefaultLegalPaths...)
	}
	return c.LegalPaths
}
