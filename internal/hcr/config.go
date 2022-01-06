package hcr

import "fmt"

type Config struct {
	PagesBranch string
	ChartsDir   string
	PreRelease  bool
	Tag         string
	Remote      string
	Token       string
	DryRun      bool
	Version     bool
}

func (c Config) String() string {
	token := "*****"
	if len(c.Token) == 0 {
		token = "<empty>"
	}
	return fmt.Sprintf("pages-branch: %q, charts-dir: %q, pre-release: %t, tag: %q, remote: %q, token: %q, dry-run: %t",
		c.PagesBranch, c.ChartsDir, c.PreRelease, c.Tag, c.Remote, token, c.DryRun)
}
