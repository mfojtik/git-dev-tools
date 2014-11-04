package repository

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mfojtik/git-dev-tools/git"
)

type Repository struct {
	Path    string
	Name    string
	Changes []git.Commit
}

// InitRepository initialize the repository based on the path
func NewRepository(path string) *Repository {
	return &Repository{
		Path: filepath.Clean(path),
		Name: filepath.Base(path),
	}
}

// Update fetches new commits from the upstream repository and merge them into
// origin/master branch. Then it will push the updated origin/master to remote
// origin repository (to fork) and checkout back the original branch.
func (r *Repository) Update() error {
	currentBranch, err := r.CurrentBranchName()
	if err != nil {
		return err
	}
	oldRef, err := r.CurrentRef()
	if err != nil {
		return err
	}
	if currentBranch != "master" {
		if out, err := r.Git("checkout", "master"); err != nil {
			return fmt.Errorf("Unable to checkout the master branch (%v):\n%v", err, out)
		}
	}
	defer func() {
		if currentBranch != "master" {
			r.Git("checkout", currentBranch)
		}
	}()
	if out, err := r.Git("fetch", "upstream"); err != nil {
		return fmt.Errorf("Unable to fetch commits from upstream (%v):\n%v", err, out)
	}
	if out, err := r.Git("merge", "upstream/master"); err != nil {
		return fmt.Errorf("Unable to merge commits from upstream (%v):\n%v", err, out)
	}
	if out, err := r.Git("push", "origin", "master"); err != nil {
		return fmt.Errorf("Unable to push commits to remote fork (%v):\n%v", err, out)
	}

	// Get the list of changes after update
	r.Changes = git.ListChanges(r.Path, oldRef, "HEAD")

	return nil
}

// currentBranch returns the name of the local branch that is currently
// checkouted.
func (r *Repository) CurrentBranchName() (string, error) {
	out, err := r.Git("rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(out), err
}

// Branches lists all local branches
func (r *Repository) Branches() []string {
	branches := []string{}
	out, _ := r.Git("branch", "--no-color")
	for _, name := range strings.Split(out, "\n") {
		if len(strings.TrimSpace(name)) == 0 {
			continue
		}
		branches = append(branches, strings.TrimSpace(strings.Replace(name, "*", "", -1)))
	}
	return branches
}

func (r *Repository) CurrentRef() (string, error) {
	out, err := r.Git("rev-parse", "--short", "HEAD")
	return strings.TrimSpace(out), err
}

// ListPushedLocalBranches lists all local branches that contains commits which
// are already pushed into upstream/master
func (r *Repository) ListPushedLocalBranches() ([]string, error) {
	defer func() {
		r.Git("checkout", "master")
	}()
	branches := []string{}
	for _, name := range r.Branches() {
		if name == "master" {
			continue
		}
		if _, err := r.Git("checkout", name); err != nil {
			return branches, fmt.Errorf("Failed to checkout %s: %v", name, err)
		}
		if out, err := r.Git("cherry", "upstream/master"); len(out) == 0 && err == nil {
			branches = append(branches, name)
		}
	}
	return branches, nil
}

// CleanBranch remove the local and remote branch
func (r *Repository) CleanBranch(name string) error {
	if out, err := r.Git("branch", "-D", name); err != nil {
		return fmt.Errorf("Unable to remove local branch '%s' (%v):\n%v", name, err, out)
	}
	if out, err := r.Git("push", "origin", ":"+name); err != nil {
		return fmt.Errorf("Unable to remove remote branch 'origin/%s' (%v):\n%v", name, err, out)
	}
	return nil
}

func (r *Repository) Git(args ...string) (string, error) {
	return git.Git(r.Path, args...)
}
