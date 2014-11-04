package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Commit struct {
	Ref     string
	Author  string
	Message string
}

// Git wraps the 'git' command execution
func Git(path string, args ...string) (string, error) {
	out := bytes.Buffer{}

	cmd := exec.Command("git", args...)
	cmd.Dir = path
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

func ListChanges(repoPath, fromRef, toRef string) []Commit {
	result := []Commit{}
	r := fmt.Sprintf("%s..%s", fromRef, toRef)
	out, err := Git(repoPath, "log", "--no-merges", "--pretty", "format='%h;%ae;%s'", r)
	if err != nil {
		return result
	}
	for _, line := range strings.Split("\n", out) {
		line := strings.TrimSpace(line)
		fields := strings.Split(";", line)
		result = append(result, Commit{fields[0], fields[1], fields[2]})
	}
	return result
}
