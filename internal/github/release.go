package github

type Release struct {
	Owner string
	Repo  string
	Tag   string

	Name        string
	Description string
	AssetPath   string
	PreRelease  bool
}
