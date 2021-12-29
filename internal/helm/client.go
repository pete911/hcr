package helm

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/provenance"
	"helm.sh/helm/v3/pkg/repo"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	pkg *action.Package
	log *zap.Logger
}

func NewClient(log *zap.Logger) Client {
	return Client{log: log, pkg: action.NewPackage()}
}

func (c Client) PackageCharts(chartsDir string) (map[string]*chart.Chart, error) {
	files, err := ioutil.ReadDir(chartsDir)
	if err != nil {
		return nil, fmt.Errorf("read charts directory: %w", err)
	}

	chs := make(map[string]*chart.Chart)
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		path, ch, err := c.PackageChart(filepath.Join(chartsDir, file.Name()))
		if err != nil {
			return nil, err
		}
		chs[path] = ch
	}
	return chs, nil
}

// PackageChart package given chart in current working directory (<name>-<version>.tgz) and return packaged chart
// location and metadata
func (c Client) PackageChart(path string) (string, *chart.Chart, error) {
	c.log.Info(fmt.Sprintf("start package %s chart", path))
	packagedChart, err := c.pkg.Run(path, nil)
	if err != nil {
		return "", nil, fmt.Errorf("package chart at %s path: %w", path, err)
	}
	c.log.Info(fmt.Sprintf("chart %s packaged as %s", path, packagedChart))
	ch, err := loader.LoadFile(packagedChart)
	if err != nil {
		return "", nil, fmt.Errorf("load chart: %w", err)
	}
	c.log.Info(fmt.Sprintf("chart %s loaded", ch.Name()))
	return packagedChart, ch, nil
}

// UpdateIndex at the specified location with given chart. Base URL is url without chart name.
func (c Client) UpdateIndex(indexFilePath, archiveChartPath string, chart *chart.Chart, downloadUrl string) (bool, error) {
	indexFile, err := c.loadIndexFile(indexFilePath)
	if err != nil {
		return false, err
	}

	// chart already exists in the index
	if _, err := indexFile.Get(chart.Name(), chart.Metadata.Version); err == nil {
		c.log.Info(fmt.Sprintf("index file get %s %s: %v", chart.Name(), chart.Metadata.Version, err))
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
