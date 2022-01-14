package hcr

import (
	"context"
	"fmt"
	"github.com/pete911/hcr/internal/git"
	"github.com/pete911/hcr/internal/github"
	"github.com/pete911/hcr/internal/helm"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"os"
	"path/filepath"
)

type Releaser struct {
	gitClient        git.Client
	ghClient         github.Client
	helmClient       helm.Client
	ghPagesDir       string
	ghPagesIndexPath string
	config           Config
	log              *zap.Logger
}

func NewReleaser(log *zap.Logger, config Config) (Releaser, error) {
	ghPagesDir, err := os.MkdirTemp("", "gh-pages")
	if err != nil {
		return Releaser{}, fmt.Errorf("create gh-pages tmp dir: %w", err)
	}
	return Releaser{
		gitClient:        git.NewClient(log),
		ghClient:         github.NewClient(log, config.Token),
		helmClient:       helm.NewClient(log, config.HelmConfig),
		ghPagesDir:       ghPagesDir,
		ghPagesIndexPath: filepath.Join(ghPagesDir, "index.yaml"),
		config:           config,
		log:              log,
	}, nil
}

func (r Releaser) Release(ctx context.Context) (map[string]*chart.Chart, error) {
	// check if the remote GitHub pages branch exists
	if err := r.pagesRemoteBranchExists(); err != nil {
		return nil, err
	}
	r.log.Info("github pages remote branch exists")

	// package charts
	charts, chartsCleanup, err := r.helmClient.PackageCharts(r.config.ChartsDir)
	if err != nil {
		return nil, fmt.Errorf("package charts: %w", err)
	}
	defer chartsCleanup()
	r.log.Info("charts packaged")

	// add GitHub pages worktree (so we can update index)
	worktreeCleanup, err := r.addPagesWorktree()
	if err != nil {
		return nil, err
	}
	defer worktreeCleanup()

	// release charts and update index
	ok, err := r.releaseChartsAndUpdateIndex(ctx, charts)
	if err != nil {
		return nil, err
	}
	if !ok {
		r.log.Info("no chart changes")
		return nil, nil
	}
	r.log.Info("released charts and updated index")

	// commit and push index
	if err := r.gitClient.AddAndCommit(r.ghPagesDir, "index.yaml", "update index.yaml"); err != nil {
		return nil, fmt.Errorf("git commit index to github pages: %w", err)
	}
	if err := r.gitClient.Push(r.ghPagesDir, r.config.Remote, r.config.PagesBranch, r.config.Token); err != nil {
		return nil, fmt.Errorf("git push github pages: %w", err)
	}

	r.log.Info("index updated and pushed to github pages")
	return charts, nil
}

// releaseChartsAndUpdateIndex releases helm chart as GitHub releases and updates index file in GitHub pages. If the charts
// were updated, true and nil error is returned. If there are no charts updated, false and nil error is returned.
// This method does not commit and push gh pages index file, only updates it.
func (r Releaser) releaseChartsAndUpdateIndex(ctx context.Context, charts map[string]*chart.Chart) (bool, error) {
	var updated bool
	for chPath, ch := range charts {
		ok, err := r.releaseChartAndUpdateIndex(ctx, chPath, ch)
		if err != nil {
			return false, err
		}
		if ok {
			updated = true
		}
	}
	return updated, nil
}

func (r Releaser) releaseChartAndUpdateIndex(ctx context.Context, chartPath string, ch *chart.Chart) (bool, error) {
	owner, repo, err := r.gitClient.GetOwnerAndRepo(r.ghPagesDir, r.config.Remote)
	if err != nil {
		return false, fmt.Errorf("get github owner and repo: %w", err)
	}

	tag := r.GetReleaseTag(ch)
	ok, err := r.ghClient.ReleaseExists(ctx, owner, repo, tag)
	if err != nil {
		return false, fmt.Errorf("%s release %s exists: %w", ch.Name(), tag, err)
	}
	if ok {
		r.log.Info(fmt.Sprintf("%s release %s already exists, skipping", ch.Name(), tag))
		return false, nil
	}
	if r.config.DryRun {
		r.log.Info(fmt.Sprintf("%s release %s skipping, dry-run set to true", ch.Name(), tag))
		r.log.Info(fmt.Sprintf("update %s index skipping, dry-run set to true", r.ghPagesIndexPath))
		return false, nil
	}

	release := github.Release{
		Name:        fmt.Sprintf("%s-%s", ch.Name(), ch.Metadata.Version),
		Description: fmt.Sprintf("Kubernetes %s Helm chart", ch.Name()),
		AssetPath:   chartPath,
		PreRelease:  r.config.PreRelease,
	}
	assetUrl, err := r.ghClient.CreateRelease(ctx, owner, repo, tag, release)
	if err != nil {
		return false, fmt.Errorf("create %s release: %w", tag, err)
	}

	ok, err = r.helmClient.UpdateIndex(r.ghPagesIndexPath, chartPath, ch, assetUrl)
	if err != nil {
		return false, fmt.Errorf("update %s index file: %w", r.ghPagesIndexPath, err)
	}
	return ok, nil
}

func (r Releaser) GetReleaseTag(ch *chart.Chart) string {
	if r.config.Tag != "" {
		return r.config.Tag
	}
	return ch.Metadata.Version
}

func (r Releaser) addPagesWorktree() (cleanup func(), err error) {
	if err := r.gitClient.AddWorktree(r.ghPagesDir, r.config.Remote, r.config.PagesBranch); err != nil {
		return nil, fmt.Errorf("add gh-pages worktree: %w", err)
	}
	r.log.Info(fmt.Sprintf("added github pages %s worktree to %s", r.config.PagesBranch, r.ghPagesDir))

	cleanup = func() {
		if err := r.gitClient.RemoveWorktree(r.ghPagesDir); err != nil {
			r.log.Error(fmt.Sprintf("remove github pages %s worktree %s: %v", r.config.PagesBranch, r.ghPagesDir, err))
			return
		}
		r.log.Info(fmt.Sprintf("removed github pages %s worktree %s", r.config.PagesBranch, r.ghPagesDir))
	}
	return cleanup, nil
}

func (r Releaser) pagesRemoteBranchExists() error {
	remoteBranches, err := r.gitClient.ListRemoteBranches(r.config.Remote)
	if err != nil {
		return fmt.Errorf("list remote branches: %w", err)
	}

	for _, remoteBranch := range remoteBranches {
		if r.config.PagesBranch == remoteBranch {
			r.log.Info(fmt.Sprintf("found %s github pages remote branch", r.config.PagesBranch))
			return nil
		}
	}
	r.log.Warn(createGHPagesMessage(r.config.PagesBranch, r.config.Remote))
	return fmt.Errorf("github pages remote branch %s does not exist", r.config.PagesBranch)
}

func createGHPagesMessage(branch, remote string) string {
	msg := `branch %s does not exist, run the following to create pages branch:
git checkout --orphan %s
git rm -rf .
git commit -m "initial commit" --allow-empty
git push -u %s %s`
	return fmt.Sprintf(msg, branch, branch, remote, branch)
}
