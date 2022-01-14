package flag

import (
	"errors"
	"flag"
	"github.com/pete911/hcr/internal/hcr"
	"github.com/pete911/hcr/internal/helm"
	"os"
	"strconv"
)

type flags struct {
	pagesBranch        string
	chartsDir          string
	helmSign           bool
	helmKey            string
	helmKeyring        string
	helmPassphraseFile string
	preRelease         bool
	tag                string
	remote             string
	token              string
	dryRun             bool
	version            bool
}

func ParseFlags() (hcr.Config, error) {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	var f flags

	flagSet.StringVar(&f.pagesBranch, "pages-branch", getStringEnv("HCR_PAGES_BRANCH", "gh-pages"), "The GitHub pages branch")
	flagSet.StringVar(&f.chartsDir, "charts-dir", getStringEnv("HCR_CHARTS_DIR", "charts"), "The Helm charts location, can be specific chart")
	flagSet.BoolVar(&f.helmSign, "helm-sign", getBoolEnv("HCR_HELM_SIGN", false), "Use a PGP private key to sign this package")
	flagSet.StringVar(&f.helmKey, "helm-key", getStringEnv("HCR_HELM_KEY", ""), "Name of the key to use when signing. Used if --sign is true")
	flagSet.StringVar(&f.helmKeyring, "helm-keyring", getStringEnv("HCR_HELM_KEYRING", ""), "Location of a public keyring")
	flagSet.StringVar(&f.helmKeyring, "helm-passphrase-file", getStringEnv("HCR_HELM_PASSPHRASE_FILE", ""), "Location of a file which contains the passphrase for the signing key")
	flagSet.BoolVar(&f.preRelease, "pre-release", getBoolEnv("HCR_PRE_RELEASE", false), "Whether the (chart) release should be marked as pre-release")
	flagSet.StringVar(&f.tag, "tag", getStringEnv("HCR_TAG", ""), "Release tag, defaults to chart version")
	flagSet.StringVar(&f.remote, "remote", getStringEnv("HCR_REMOTE", "origin"), "The Git remote for the GitHub Pages branch")
	flagSet.StringVar(&f.token, "token", getStringEnv("HCR_TOKEN", ""), "GitHub Auth Token")
	flagSet.BoolVar(&f.dryRun, "dry-run", getBoolEnv("HCR_DRY_RUN", false), "Whether to skip release update gh-pages index update")
	flagSet.BoolVar(&f.version, "version", getBoolEnv("HCR_VERSION", false), "Print hcr version")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return hcr.Config{}, err
	}

	if err := f.validate(); err != nil {
		return hcr.Config{}, err
	}

	helmConfig := helm.Config{
		Sign:           f.helmSign,
		Key:            f.helmKey,
		Keyring:        f.helmKeyring,
		PassphraseFile: f.helmPassphraseFile,
	}

	return hcr.Config{
		PagesBranch: f.pagesBranch,
		ChartsDir:   f.chartsDir,
		HelmConfig:  helmConfig,
		PreRelease:  f.preRelease,
		Tag:         f.tag,
		Remote:      f.remote,
		Token:       f.token,
		DryRun:      f.dryRun,
		Version:     f.version,
	}, nil
}

func (f flags) validate() error {
	if f.pagesBranch == "" {
		return errors.New("pages-branch cannot be empty")
	}
	if f.chartsDir == "" {
		return errors.New("charts-dir cannot be empty")
	}
	if f.remote == "" {
		return errors.New("remote cannot be empty")
	}
	return nil
}

func getStringEnv(envName string, defaultValue string) string {
	env, ok := os.LookupEnv(envName)
	if !ok {
		return defaultValue
	}
	return env
}

func getBoolEnv(envName string, defaultValue bool) bool {
	env, ok := os.LookupEnv(envName)
	if !ok {
		return defaultValue
	}

	if v, err := strconv.ParseBool(env); err == nil {
		return v
	}
	return defaultValue
}
