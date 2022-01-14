package hcr

import (
	"fmt"
	"github.com/pete911/hcr/internal/helm"
	"github.com/pete911/hcr/internal/utils"
)

type Config struct {
	PagesBranch string
	ChartsDir   string
	HelmConfig  helm.Config
	PreRelease  bool
	Tag         string
	Remote      string
	Token       string
	DryRun      bool
	Version     bool
}

func (c Config) String() string {
	return fmt.Sprintf("pages-branch: %q, charts-dir: %q, pre-release: %t, tag: %q, remote: %q, token: %s, dry-run: %t, helm-config: %s",
		c.PagesBranch, c.ChartsDir, c.PreRelease, c.Tag, c.Remote, utils.SecretValue(c.Token), c.DryRun, c.HelmConfig)
}
