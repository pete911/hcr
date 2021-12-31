package hcr

import "fmt"

type Config struct {
	PagesBranch string
	ChartsDir   string
	Tag         string
	Remote      string
	Token       string
}

func (c Config) String() string {
	token := "*****"
	if len(c.Token) == 0 {
		token = "<empty>"
	}
	return fmt.Sprintf("pages-branch: %q, charts-dir: %q, tag: %q, remote: %q, token: %q", c.PagesBranch, c.ChartsDir, c.Tag, c.Remote, token)
}
