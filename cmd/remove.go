package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Remove(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: github-proxy remove <owner>/<repo>\n")
		os.Exit(1)
	}

	repoRef := args[0]
	parts := strings.SplitN(repoRef, "/", 2)
	if len(parts) != 2 {
		fmt.Fprintf(os.Stderr, "ERROR: expected format <owner>/<repo>\n")
		os.Exit(1)
	}

	owner, repo := parts[0], parts[1]

	dir := dataDirPath()
	reposDir := filepath.Join(dir, "repos")
	repoPath := filepath.Join(reposDir, owner, repo+".git")

	info, err := os.Lstat(repoPath)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "ERROR: repo not registered: %s/%s\n", owner, repo)
		os.Exit(1)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(repoPath)
		if err != nil {
			target = "<unknown>"
		}
		fmt.Printf("Removing symlink to %s\n", target)
	} else {
		fmt.Printf("Removing bare clone\n")
	}

	if err := os.RemoveAll(repoPath); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: removing repo: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Removed: %s/%s\n", owner, repo)

	ownerDir := filepath.Join(reposDir, owner)
	remaining, _ := os.ReadDir(ownerDir)
	if len(remaining) == 0 {
		os.Remove(ownerDir)
	}
}
