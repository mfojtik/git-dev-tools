package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/op/go-logging"
)

var (
	log    = logging.MustGetLogger("git-sync")
	format = "%{color} ▶ %{level:.4s} %{color:reset} %{message}"
)

type Repository struct {
	Path string
}

func InitRepository(path string) *Repository {
	return &Repository{Path: filepath.Clean(path)}
}

func (r *Repository) Name() string {
	return filepath.Base(r.Path)
}

func (r *Repository) Update() error {
	if out, err := r.Git("checkout", "master"); err != nil {
		return fmt.Errorf("Unable to checkout the master branch (%v):\n%v", err, out)
	}
	if out, err := r.Git("fetch", "upstream"); err != nil {
		return fmt.Errorf("Unable to fetch commits from upstream (%v):\n%v", err, out)
	}
	if out, err := r.Git("merge", "upstream/master"); err != nil {
		return fmt.Errorf("Unable to merge commits from upstream (%v):\n%v", err, out)
	}
	return nil
}

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
	out := bytes.Buffer{}

	cmd := exec.Command("git", args...)
	cmd.Dir = r.Path
	cmd.Stderr = &out
	cmd.Stdout = &out

	if err := cmd.Start(); err != nil {
		return out.String(), err
	}

	if err := cmd.Wait(); err != nil {
		return out.String(), err
	}
	return out.String(), nil
}

func readGitReposFile(path string) ([]string, error) {
	gitDirectories := []string{}
	file, err := os.Open(path + "/.gitrepos")
	defer file.Close()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		gitDirectories = append(
			gitDirectories,
			filepath.Clean(fmt.Sprintf("%s/%s", path, strings.TrimSpace(scanner.Text()))),
		)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return gitDirectories, nil
}

func main() {
	logging.SetFormatter(logging.MustStringFormatter(format))
	flag.Parse()

	if flag.Arg(0) == "" {
		log.Critical("No directory specified. Quitting.")
		os.Exit(1)
	}

	repos, err := readGitReposFile(flag.Arg(0))
	if err != nil {
		log.Critical("Unable to read %s/.gitrepos file. Aborting.", flag.Arg(0))
		os.Exit(1)
	}

	var (
		syncGroup  sync.WaitGroup
		cleanGroup sync.WaitGroup
	)

	for _, path := range repos {
		syncGroup.Add(1)
		go func(repoPath string) {
			defer syncGroup.Done()
			r := InitRepository(repoPath)
			if err := r.Update(); err != nil {
				log.Error("Repository '%v' failed to update: %v", r.Name(), err.Error())
				return
			} else {
				log.Info("Repository '%v' successfully updated", r.Name())
			}
			if cleanupBranches, err := r.ListPushedLocalBranches(); err != nil {
				log.Error("Failed to get list of pushed branches for %s: %v", r.Name(), err.Error())
				return
			} else {
				if len(cleanupBranches) == 0 {
					return
				}
				log.Info("Cleaning up %d branches for %s [%v]", len(cleanupBranches), r.Name(), cleanupBranches)
				for _, name := range cleanupBranches {
					cleanGroup.Add(1)
					go func(branchName string, repo *Repository) {
						if err := repo.CleanBranch(branchName); err != nil {
							log.Error("Failed to cleanup '%s' branch in '%s' repository: %v", branchName, repo.Name(), err.Error())
						}
					}(name, r)
				}
			}
		}(path)
	}

	cleanGroup.Wait()
	syncGroup.Wait()

}
