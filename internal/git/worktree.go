// Package git provides the minimal git operations xocode needs to isolate a
// build in a worktree. Worktree creation itself is delegated to
// `cursor-agent -w`; this package handles repo detection, naming, and the
// human-facing review/merge guidance.
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsRepo reports whether dir is inside a git work tree.
func IsRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// Init initializes a new git repository at dir and makes an initial empty
// commit so a worktree has a base to branch from.
func Init(dir string) error {
	if err := run(dir, "init"); err != nil {
		return err
	}
	// An initial commit gives `git worktree add` a HEAD to base on. Allow it to
	// be empty so we don't force-add the user's files.
	_ = run(dir, "add", "-A")
	if err := run(dir, "commit", "-m", "Initial commit", "--allow-empty"); err != nil {
		return err
	}
	return nil
}

// RepoName returns the basename of the repository top level, matching what
// `cursor-agent -w` uses under ~/.cursor/worktrees/<repo>/.
func RepoName(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return filepath.Base(dir)
	}
	return filepath.Base(strings.TrimSpace(string(out)))
}

// WorktreeName builds a collision-resistant worktree name from a slug and a
// short time component.
func WorktreeName(slug, shortTime string) string {
	if slug == "" {
		slug = "task"
	}
	return fmt.Sprintf("xocode-%s-%s", slug, shortTime)
}

// WorktreePath is where `cursor-agent -w <name>` places the worktree.
func WorktreePath(repo, name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	return filepath.Join(home, ".cursor", "worktrees", repo, name)
}

// MergeGuidance returns copy-pasteable commands for reviewing, merging, or
// discarding the build worktree.
func MergeGuidance(repoDir, worktreePath string) []string {
	return []string{
		fmt.Sprintf("Review:  cd %q && git status && git diff", worktreePath),
		fmt.Sprintf("Branch:  git -C %q branch --show-current", worktreePath),
		fmt.Sprintf("Merge:   git -C %q merge --no-ff $(git -C %q branch --show-current)", repoDir, worktreePath),
		fmt.Sprintf("Discard: git -C %q worktree remove %q", repoDir, worktreePath),
	}
}

func run(dir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
