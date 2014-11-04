package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mfojtik/git-dev-tools/repository"
	"github.com/op/go-logging"
)

const GitRepoFileName = ".gitrepos"

// Setup logging
var (
	log    = logging.MustGetLogger("git-sync")
	format = "%{color} ▶ %{level:.4s} %{color:reset} %{message}"
)

// readGitReposFile reads the '.gitrepos' file which contains list of GIT
// repositories we manage
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

func reportChanges(r *repository.Repository) {
	if len(r.Changes) == 0 {
		return
	}
	fmt.Printf("Changes for %s\n\n", r.Name)
	for _, c := range r.Changes {
		fmt.Printf("%s (by %s)\n", c.Message, c.Author)
	}
	fmt.Println()
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
		syncGroup    sync.WaitGroup
		cleanGroup   sync.WaitGroup
		repositories []*repository.Repository
	)

	// The main sync routine will do following:
	// Step 1: Update the repository
	// Step 2: Check if the repository contains branches that are already pushed
	// Step 3: Remove these branches
	for _, path := range repos {
		syncGroup.Add(1)
		go func(repoPath string) {
			defer syncGroup.Done()
			r := repository.NewRepository(repoPath)
			if err := r.Update(); err != nil {
				log.Error("Repository '%v' failed to update: %v", r.Name, err.Error())
				return
			} else {
				log.Info("Repository '%v' successfully updated", r.Name)
				repositories = append(repositories, r)
			}
			if cleanupBranches, err := r.ListPushedLocalBranches(); err != nil {
				log.Error("Failed to get list of pushed branches for %s: %v", r.Name, err.Error())
				return
			} else {
				if len(cleanupBranches) == 0 {
					return
				}
				log.Info("Cleaning up %d branches for %s [%v]", len(cleanupBranches), r.Name, cleanupBranches)
				for _, name := range cleanupBranches {
					cleanGroup.Add(1)
					go func(branchName string, repo *repository.Repository) {
						if err := repo.CleanBranch(branchName); err != nil {
							log.Error("Failed to cleanup '%s' branch in '%s' repository: %v", branchName, repo.Name, err.Error())
						}
					}(name, r)
				}
			}
		}(path)
	}

	cleanGroup.Wait()
	syncGroup.Wait()

	// After all operations completed, report all changes...
	for _, r := range repositories {
		reportChanges(r)
	}
}
