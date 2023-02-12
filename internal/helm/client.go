package helm

import (
	"errors"
	"fmt"
	"github.com/pete911/hcr/internal/utils"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/provenance"
	"helm.sh/helm/v3/pkg/repo"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Sign           bool
	Key            string
	Keyring        string
	PassphraseFile string
}

func (c Config) String() string {
	return fmt.Sprintf("sign: %t, key: %s, keyring: %s, passphrase-file: %s",
		c.Sign, utils.SecretValue(c.Key), utils.SecretValue(c.Keyring), utils.SecretValue(c.PassphraseFile))
}

type Client struct {
	pkg *action.Package
	log *zap.Logger
}

func NewClient(log *zap.Logger, config Config) Client {
	return Client{
		pkg: &action.Package{
			Sign:           config.Sign,
			Key:            config.Key,
			Keyring:        config.Keyring,
			PassphraseFile: config.PassphraseFile,
		},
		log: log,
	}
}

func (c Client) PackageCharts(chartsDir string) (charts map[string]*chart.Chart, cleanup func(), err error) {
	if stat, err := os.Stat(chartsDir); err != nil || !stat.IsDir() {
		return nil, nil, fmt.Errorf("charts dir %s does not exist", chartsDir)
	}

	chartsPaths, err := c.getChartsPaths(chartsDir)
	if err != nil {
		return nil, nil, err
	}

	var packagedChartsPaths []string
	chs := make(map[string]*chart.Chart)
	for _, chartPath := range chartsPaths {
		packagedChartPath, ch, err := c.PackageChart(chartPath)
		if err != nil {
			return nil, nil, err
		}
		packagedChartsPaths = append(packagedChartsPaths, packagedChartPath)
		chs[packagedChartPath] = ch
	}

	cleanup = func() {
		for _, packagedChartPath := range packagedChartsPaths {
			if err := os.Remove(packagedChartPath); err != nil {
				c.log.Warn(fmt.Sprintf("remove %s chart: %v", packagedChartPath, err))
			}
			c.log.Info(fmt.Sprintf("removed generated chart %s", packagedChartPath))
		}
	}
	return chs, cleanup, nil
}

// PackageChart package given chart in current working directory (<name>-<version>.tgz) and return packaged chart
// path and metadata
func (c Client) PackageChart(chartPath string) (string, *chart.Chart, error) {
	c.log.Info(fmt.Sprintf("start package %s chart", chartPath))
	packagedChartPath, err := c.pkg.Run(chartPath, nil)
	if err != nil {
		return "", nil, fmt.Errorf("package chart at %s path: %w", chartPath, err)
	}
	c.log.Info(fmt.Sprintf("chart %s packaged as %s", chartPath, packagedChartPath))
	ch, err := loader.LoadFile(packagedChartPath)
	if err != nil {
		return "", nil, fmt.Errorf("load chart: %w", err)
	}
	c.log.Info(fmt.Sprintf("chart %s loaded", ch.Name()))
	return packagedChartPath, ch, nil
}

// UpdateIndex at the specified location with given chart. Base URL is url without chart name.
func (c Client) UpdateIndex(indexFilePath, archiveChartPath string, chart *chart.Chart, downloadUrl string) (bool, error) {
	indexFile, err := c.loadIndexFile(indexFilePath)
	if err != nil {
		return false, err
	}

	// chart already exists in the index
	if _, err := indexFile.Get(chart.Name(), chart.Metadata.Version); err == nil {
		c.log.Info(fmt.Sprintf("chart %s %s already exists in the helm index", chart.Name(), chart.Metadata.Version))
		return false, nil
	}

	digest, err := provenance.DigestFile(archiveChartPath)
	if err != nil {
		return false, fmt.Errorf("calculate chart sha256 digest: %w", err)
	}

	baseUrl := strings.TrimSuffix(downloadUrl, archiveChartPath)
	if err := indexFile.MustAdd(chart.Metadata, archiveChartPath, baseUrl, digest); err != nil {
		return false, err
	}

	indexFile.SortEntries()
	indexFile.Generated = time.Now()

	if err := indexFile.WriteFile(indexFilePath, 0644); err != nil {
		return false, err
	}
	return true, nil
}

// loadIndexFile loads index file from specified file path, if the file does not exist, new index is returned
func (c Client) loadIndexFile(filePath string) (*repo.IndexFile, error) {
	if _, err := os.Stat(filePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.log.Info(fmt.Sprintf("creating new index file, %s does not exist", filePath))
			return repo.NewIndexFile(), nil
		}
		return nil, err
	}

	indexFile, err := repo.LoadIndexFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("load index file %s: %w", filePath, err)
	}
	c.log.Info(fmt.Sprintf("loaded %s index file", filePath))
	return indexFile, nil
}

// getChartsPaths walks supplied charts dir recursively and returns parent directories of all 'Chart.yaml' files
func (c Client) getChartsPaths(chartsDir string) ([]string, error) {
	var paths []string
	if err := filepath.WalkDir(chartsDir, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && d.Name() == "Chart.yaml" {
			paths = append(paths, filepath.Dir(path))
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return paths, nil
}
