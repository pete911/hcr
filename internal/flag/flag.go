package flag

import (
	"errors"
	"flag"
	"github.com/pete911/hcr/internal/hcr"
	"os"
)

type flags struct {
	pagesBranch string
	chartsDir   string
	remote      string
	token       string
}

func ParseFlags() (hcr.Config, error) {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	var f flags

	flagSet.StringVar(&f.pagesBranch, "pages-branch", getStringEnv("HCR_PAGES_BRANCH", "gh-pages"), "The GitHub pages branch")
	flagSet.StringVar(&f.chartsDir, "charts-dir", getStringEnv("HCR_CHARTS_DIR", "charts"), "The Helm charts location")
	flagSet.StringVar(&f.remote, "remote", getStringEnv("HCR_REMOTE", "origin"), "The Git remote for the GitHub Pages branch")
	flagSet.StringVar(&f.token, "token", getStringEnv("HCR_TOKEN", ""), "GitHub Auth Token")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return hcr.Config{}, err
	}

	if err := f.validate(); err != nil {
		return hcr.Config{}, err
	}

	return hcr.Config{
		PagesBranch: f.pagesBranch,
		ChartsDir:   f.chartsDir,
		Remote:      f.remote,
		Token:       f.token,
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
