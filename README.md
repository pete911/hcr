# hcr
Helm chart releaser is a tool to help to host helm repository as GitHub page, where packaged chart is uploaded as
a release asset and the index file is hosted on GitHub page. This tool is very similar to
[helm chart releaser](https://github.com/helm/chart-releaser), but is simpler and works with private repos as well.

## Download
- [binary](https://github.com/pete911/hcr/releases)

## Install

### Brew
- add tap `brew tap pete911/tap`
- install `brew install hcr`

### Go
- clone `git clone https://github.com/pete911/hcr.git && cd hcr`
- install `go install`
```

## Use
```
Usage of hcr:
  -charts-dir string
        The Helm charts location (default "charts")
  -dry-run
        Whether to skip release update gh-pages index update
  -pages-branch string
        The GitHub pages branch (default "gh-pages")
  -pre-release
        Whether the (chart) release should be marked as pre-release
  -remote string
        The Git remote for the GitHub Pages branch (default "origin")
  -tag string
        Release tag, defaults to chart version
  -token string
        GitHub Auth Token
  -version
        Print hcr version
```

Simply run `hcr` inside the project, this will:
- check if `-pages-branch` exists
- packages helm charts from `-charts-dir` to current directory as `<name>-<version>.tgz`
- adds `-pages-branch` git worktree to temp. directory
- creates release per chart in `-charts-dir` and uploads packaged chart as asset to that release
- update index file with the released charts, commit and push index file to `-pages-branch`

This makes it simpler and easier than [helm chart releaser](https://github.com/helm/chart-releaser), because we are not
reading from GitHub pages (or downloading releases) over http, so we don't face issues with restrictions on private
repositories (autogenerated pages link, authentication, ...), caching (GitHub pages are not updated immediately with
the latest changes) etc.

## Release
Releases are published when the new tag is created e.g. git tag -m "add super cool feature" v0.0.1 && git push --follow-tags
