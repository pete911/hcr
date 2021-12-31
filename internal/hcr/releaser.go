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
	gitClient  git.Client
	ghClient   github.Client
	helmClient helm.Client
	config     Config
	log        *zap.Logger
}

func NewReleaser(log *zap.Logger, config Config) Releaser {
	return Releaser{
		gitClient:  git.NewClient(log),
		ghClient:   github.NewClient(log, config.Token),
		helmClient: helm.NewClient(log),
		config:     config,
		log:        log,
	}
}

func (r Releaser) Release(ctx context.Context) error {
	// check if the remote GitHub pages branch exists
	if err := r.pagesRemoteBranchExists(); err != nil {
		return err
	}
	r.log.Info("github pages remote branch exists")

	// package charts
	charts, err := r.helmClient.PackageCharts(r.config.ChartsDir)
	if err != nil {
		return fmt.Errorf("package charts: %w", err)
	}
	r.log.Info("charts packaged")

	// add GitHub pages worktree (so we can update index)
	ghPagesDir, cleanup, err := r.addPagesWorktree()
	if err != nil {
		return err
	}
	defer cleanup()

	// release charts and update index
	ok, err := r.releaseAndUpdateIndex(ctx, ghPagesDir, charts)
	if err != nil {
		return err
	}
	if !ok {
		r.log.Info("no chart changes")
		return nil
	}
	r.log.Info("released charts and updated index")

	// commit and push index
	if err := r.gitClient.AddAndCommit(ghPagesDir, "index.yaml", "update index.yaml"); err != nil {
		return fmt.Errorf("git commit index to github pages: %w", err)
	}
	if err := r.gitClient.Push(ghPagesDir, r.config.Remote, r.config.PagesBranch, r.config.Token); err != nil {
		return fmt.Errorf("git push github pages: %w", err)
	}

	r.log.Info("index updated and pushed to github pages")
	return nil
}

// releaseAndUpdateIndex releases helm chart as GitHub releases and updates index file in GitHub pages. If the charts
// were updated, true and nil error is returned. If there are no charts updated, false and nil error is returned.
// This method does not commit and push gh pages index file, only updates it.
func (r Releaser) releaseAndUpdateIndex(ctx context.Context, ghPagesDir string, charts map[string]*chart.Chart) (bool, error) {
	owner, repo, err := r.gitClient.GetOwnerAndRepo(ghPagesDir, r.config.Remote)
	if err != nil {
		return false, fmt.Errorf("get github owner and repo: %w", err)
	}

	var updated bool
	indexFile := filepath.Join(ghPagesDir, "index.yaml")
	for chPath, ch := range charts {
		tag := r.config.Tag
		if tag == "" {
			tag = ch.Metadata.Version
		}
		ok, err := r.ghClient.ReleaseExists(ctx, owner, repo, tag)
		if err != nil {
			return false, fmt.Errorf("%s release %s exists: %w", ch.Name(), tag, err)
		}
		if ok {
			r.log.Info(fmt.Sprintf("%s release %s already exists, skipping", ch.Name(), tag))
			continue
		}

		release := github.Release{
			Name:        fmt.Sprintf("%s-%s", ch.Name(), ch.Metadata.Version),
			Description: fmt.Sprintf("Kubernetes %s Helm chart", ch.Name()),
			AssetPath:   chPath,
			PreRelease:  false,
		}
		assetUrl, err := r.ghClient.CreateRelease(ctx, owner, repo, tag, release)
		if err != nil {
			return false, fmt.Errorf("create %s release: %w", tag, err)
		}

		ok, err = r.helmClient.UpdateIndex(indexFile, chPath, ch, assetUrl)
		if err != nil {
			return false, fmt.Errorf("update %s index file: %w", indexFile, err)
		}
		if ok {
			updated = true
		}
	}
	return updated, nil
}

func (r Releaser) addPagesWorktree() (ghPagesDir string, cleanup func(), err error) {
	ghPagesDir, err = os.MkdirTemp("", "gh-pages")
	if err != nil {
		return "", nil, fmt.Errorf("create gh-pages tmp dir: %w", err)
	}
	if err := r.gitClient.AddWorktree(ghPagesDir, r.config.Remote, r.config.PagesBranch); err != nil {
		return "", nil, fmt.Errorf("add gh-pages worktree: %w", err)
	}
	r.log.Info(fmt.Sprintf("added github pages %s worktree to %s", r.config.PagesBranch, ghPagesDir))

	cleanup = func() {
		if err := r.gitClient.RemoveWorktree(ghPagesDir); err != nil {
			r.log.Error(fmt.Sprintf("remove github pages %s worktree %s: %v", r.config.PagesBranch, ghPagesDir, err))
			return
		}
		r.log.Info(fmt.Sprintf("removed github pages %s worktree %s", r.config.PagesBranch, ghPagesDir))
	}
	return ghPagesDir, cleanup, nil
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
