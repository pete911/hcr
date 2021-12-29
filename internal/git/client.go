package git

import (
	"bufio"
	"bytes"
	"fmt"
	"go.uber.org/zap"
	"os/exec"
	"strings"
)

type Client struct {
	log *zap.Logger
}

func NewClient(log *zap.Logger) Client {
	return Client{log: log}
}

func (c Client) ListRemoteBranches(remote string) ([]string, error) {
	b, err := c.cmdOutput("", exec.Command("git", "ls-remote", "--heads", remote), false)
	if err != nil {
		return nil, err
	}

	var branches []string
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		columns := strings.Fields(sc.Text())
		if len(columns) == 2 {
			branch := strings.TrimPrefix(strings.TrimSpace(columns[1]), "refs/heads/")
			branches = append(branches, branch)
		}
	}
	return branches, nil
}

func (c Client) AddWorktree(path, remote, branch string) error {
	if err := c.cmdRun("", exec.Command("git", "remote", "update", remote, "--prune"), false); err != nil {
		return err
	}
	cmt := fmt.Sprintf("%s/%s", remote, branch)
	return c.cmdRun("", exec.Command("git", "worktree", "add", path, cmt), false)
}

func (c Client) RemoveWorktree(path string) error {
	return c.cmdRun("", exec.Command("git", "worktree", "remove", path, "--force"), false)
}

func (c Client) AddAndCommit(workingDir, file, message string) error {
	if err := c.cmdRun(workingDir, exec.Command("git", "add", file), false); err != nil {
		return err
	}
	return c.cmdRun(workingDir, exec.Command("git", "commit", "-m", fmt.Sprintf("%q", message)), false)
}

func (c Client) Push(workingDir, remote, branch, token string) error {
	pushUrl, err := c.getPushUrl(workingDir, remote, token)
	if err != nil {
		return err
	}

	// run silently, so we don't log token (if it has been supplied)
	fullBranch := fmt.Sprintf("HEAD:refs/heads/%s", branch)
	return c.cmdRun(workingDir, exec.Command("git", "push", pushUrl, fullBranch), true)
}

func (c Client) GetOwnerAndRepo(workingDir, remote string) (string, string, error) {
	b, err := c.cmdOutput(workingDir, exec.Command("git", "remote", "get-url", "--push", remote), false)
	if err != nil {
		return "", "", err
	}
	return getOwnerAndRepoFromUrl(strings.TrimSpace(string(b)))
}

func (c Client) getPushUrl(workingDir, remote, token string) (string, error) {
	b, err := c.cmdOutput(workingDir, exec.Command("git", "remote", "get-url", "--push", remote), false)
	if err != nil {
		return "", err
	}
	remoteUrl := strings.TrimSpace(string(b))

	if token == "" {
		c.log.Info(fmt.Sprintf("no token supplied, returning %s push url", remoteUrl))
		return remoteUrl, nil
	}

	owner, repo, err := getOwnerAndRepoFromUrl(remoteUrl)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, owner, repo), nil
}

func getOwnerAndRepoFromUrl(remoteUrl string) (string, string, error) {
	// this can be either git@github.com:<org>/<repo>.git or https://github.com/<org>/<repo>.git
	columnParts := strings.Split(remoteUrl, ":")
	if len(columnParts) != 2 {
		return "", "", fmt.Errorf("invalid remote url %s, no : found", remoteUrl)
	}
	pathParts := strings.Split(columnParts[1], "/")
	if len(pathParts) < 2 {
		return "", "", fmt.Errorf("invalid remote url %s, min 2 path parts expected", remoteUrl)
	}
	return pathParts[len(pathParts)-2], strings.TrimSuffix(pathParts[len(pathParts)-1], ".git"), nil
}

// cmdRun runs specified command in the working dir, input and output is logged
func (c Client) cmdRun(workingDir string, cmd *exec.Cmd, silent bool) error {
	if b, err := c.cmdOutput(workingDir, cmd, silent); err != nil {
		return fmt.Errorf("%s: %w", string(b), err)
	}
	return nil
}

// cmdOutput runs specified command in the working dir, input and output is logged and output returned as well
func (c Client) cmdOutput(workingDir string, cmd *exec.Cmd, silent bool) ([]byte, error) {
	cmd.Dir = workingDir
	if !silent {
		c.log.Info(cmd.String())
	}
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", string(b), err)
	}
	return b, nil
}
