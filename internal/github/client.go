package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/v36/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"net/http"
	"os"
	"time"
)

const httpTimeout = 5 * time.Second

type Client struct {
	gh  *github.Client
	log *zap.Logger
}

// NewClient returns "logged in" GitHub client if the token is not empty.
func NewClient(log *zap.Logger, token string) Client {
	if token == "" {
		return Client{log: log, gh: github.NewClient(&http.Client{Timeout: httpTimeout})}
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	tc.Timeout = httpTimeout
	return Client{log: log, gh: github.NewClient(tc)}
}

// ReleaseAndAssetExists checks if the release and asset already exists
func (c Client) ReleaseAndAssetExists(ctx context.Context, owner, repo, tag, assetPath string) (bool, bool, error) {
	release, _, err := c.gh.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		// release does not exist (no assets)
		if ghError, ok := err.(*github.ErrorResponse); ok && ghError.Response.StatusCode == http.StatusNotFound {
			return false, false, nil
		}
	}

	for _, asset := range release.Assets {
		if asset == nil {
			continue
		}
		if asset.GetName() == assetPath {
			return true, true, nil
		}
	}
	return true, false, err
}

// CreateRelease creates release (if it doesn't exist) and returns release id
func (c Client) CreateRelease(ctx context.Context, release Release, dryRun bool) (int64, error) {
	existingRelease, _, err := c.gh.Repositories.GetReleaseByTag(ctx, release.Owner, release.Repo, release.Tag)
	if err == nil {
		c.log.Info(fmt.Sprintf("%s release %s already exists, skipping create release", release.Name, release.Tag))
		return existingRelease.GetID(), nil
	}
	if err != nil {
		// not a gitHub error and not 404
		if ghError, ok := err.(*github.ErrorResponse); !ok || ghError.Response.StatusCode != http.StatusNotFound {
			return 0, fmt.Errorf("get release by %s tag: %w", release.Tag, err)
		}
	}
	if dryRun {
		c.log.Info(fmt.Sprintf("%s create release %s skipping, dry run is set to true", release.Name, release.Tag))
		return 0, nil
	}

	request := &github.RepositoryRelease{
		Name:       &release.Name,
		Body:       &release.Description,
		TagName:    &release.Tag,
		Prerelease: &release.PreRelease,
	}

	response, _, err := c.gh.Repositories.CreateRelease(ctx, release.Owner, release.Repo, request)
	if err != nil {
		return 0, fmt.Errorf("%s create release %s: %w", release.Name, release.Tag, err)
	}
	c.log.Info(fmt.Sprintf("%s release %s with id %d created", release.Name, release.Tag, response.GetID()))
	return response.GetID(), nil
}

// UploadAsset upload asset and return asset download url
func (c Client) UploadAsset(ctx context.Context, releaseId int64, release Release) (string, error) {
	existingRelease, _, err := c.gh.Repositories.GetRelease(ctx, release.Owner, release.Repo, releaseId)
	if err != nil {
		return "", fmt.Errorf("get release by %d id: %w", releaseId, err)
	}
	for _, asset := range existingRelease.Assets {
		if asset != nil && asset.GetName() == release.AssetPath {
			c.log.Info(fmt.Sprintf("%s release %s asset %s already exists, skipping create asset", release.Name, release.Tag, asset.GetBrowserDownloadURL()))
			return asset.GetURL(), nil
		}
	}

	f, err := os.Open(release.AssetPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	opts := &github.UploadOptions{Name: release.AssetPath}
	asset, _, err := c.gh.Repositories.UploadReleaseAsset(ctx, release.Owner, release.Repo, releaseId, opts, f)
	return asset.GetURL(), nil
}
