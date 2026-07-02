package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Update(args []string) {
	var owner, repo string

	i := 0
	for i < len(args) {
		switch args[i] {
		case "--owner":
			i++
			if i < len(args) {
				owner = args[i]
			}
		case "--repo":
			i++
			if i < len(args) {
				repo = args[i]
			}
		default:
			if owner == "" && repo == "" {
				parts := strings.SplitN(args[i], "/", 2)
				if len(parts) == 2 {
					owner, repo = parts[0], parts[1]
				}
			}
		}
		i++
	}

	if owner == "" || repo == "" {
		fmt.Fprintf(os.Stderr, "Usage: github-proxy update <owner/repo> [--owner <owner>] [--repo <name>]\n")
		os.Exit(1)
	}

	dir := dataDirPath()
	destPath := filepath.Join(dir, "repos", owner, repo+".git")

	fi, err := os.Lstat(destPath)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "ERROR: repo not registered: %s/%s\n", owner, repo)
		os.Exit(1)
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		fmt.Printf("Skipping symlink: %s/%s points to a live repo on disk, no fetch needed.\n", owner, repo)
		return
	}

	cmd := exec.Command("git", "--git-dir", destPath, "fetch", "origin",
		"+refs/heads/*:refs/heads/*",
		"+refs/tags/*:refs/tags/*",
		"--prune")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: git fetch failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated: %s/%s\n", owner, repo)
}
