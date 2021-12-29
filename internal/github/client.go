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

// ReleaseExists checks if the release already exists
func (c Client) ReleaseExists(ctx context.Context, owner, repo, tag string) (bool, error) {
	_, _, err := c.gh.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err == nil {
		return true, nil
	}

	if ghError, ok := err.(*github.ErrorResponse); ok {
		if ghError.Response.StatusCode == http.StatusNotFound {
			return false, nil
		}
	}
	return false, err
}

// CreateRelease creates release and returns asset url
func (c Client) CreateRelease(ctx context.Context, owner, repo, tag string, release Release) (string, error) {

	request := &github.RepositoryRelease{
		Name:       &release.Name,
		Body:       &release.Description,
		TagName:    &tag,
		Prerelease: &release.PreRelease,
	}

	response, _, err := c.gh.Repositories.CreateRelease(ctx, owner, repo, request)
	if err != nil {
		return "", err
	}
	c.log.Info(fmt.Sprintf("release %s created", release.Name))

	// upload assets
	url, err := c.uploadAsset(ctx, owner, repo, *response.ID, release.AssetPath)
	if err != nil {
		return "", fmt.Errorf("upload release asset %s: %w", release.AssetPath, err)
	}
	c.log.Info(fmt.Sprintf("asset %s uploaded", release.AssetPath))
	return url, nil
}

func (c Client) uploadAsset(ctx context.Context, owner, repo string, id int64, assetPath string) (string, error) {
	f, err := os.Open(assetPath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := f.Close(); err != nil {
			c.log.Warn(fmt.Sprintf("close %s asset file: %v", assetPath, err))
		}
	}()

	opts := &github.UploadOptions{Name: assetPath}
	asset, _, err := c.gh.Repositories.UploadReleaseAsset(ctx, owner, repo, id, opts, f)
	return asset.GetBrowserDownloadURL(), err
}
