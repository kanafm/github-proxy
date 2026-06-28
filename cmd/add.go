package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func Add(args []string) {
	var owner, repo string
	var repoPath string

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
			if repoPath == "" {
				repoPath = args[i]
			}
		}
		i++
	}

	if repoPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: github-proxy add <path-to-repo> [--owner <owner>] [--repo <name>]\n")
		os.Exit(1)
	}

	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "ERROR: path does not exist: %s\n", repoPath)
		os.Exit(1)
	}

	if owner == "" || repo == "" {
		remoteURL := getRemoteURL(repoPath)
		if remoteURL != "" {
			owner, repo = parseGitHubURL(remoteURL)
		}
	}

	if owner == "" || repo == "" {
		fmt.Fprintf(os.Stderr, "Could not determine owner/repo from git remote.\n")
		fmt.Fprintf(os.Stderr, "Provide them manually via flags:\n")
		fmt.Fprintf(os.Stderr, "  github-proxy add <path> --owner <owner> --repo <name>\n")
		os.Exit(1)
	}

	dir := dataDirPath()
	reposDir := filepath.Join(dir, "repos")
	destPath := filepath.Join(reposDir, owner, repo+".git")

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: creating target directory: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(destPath); err == nil {
		fmt.Fprintf(os.Stderr, "ERROR: repo already registered: %s/%s\n", owner, repo)
		os.Exit(1)
	}

	switch detectRepoType(repoPath) {
	case "working-tree":
		fmt.Printf("Type: working tree → creating bare clone at %s\n", destPath)
		cmd := exec.Command("git", "clone", "--bare", repoPath, destPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: git clone --bare failed: %v\n", err)
			os.Exit(1)
		}

	case "bare":
		fmt.Printf("Type: bare repo → creating symlink at %s\n", destPath)
		absPath, err := filepath.Abs(repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: resolving path: %v\n", err)
			os.Exit(1)
		}
		if err := os.Symlink(absPath, destPath); err != nil {
			fmt.Fprintf(os.Stderr, "Creating bare clone instead (symlink failed: %v)\n", err)
			cmd := exec.Command("git", "clone", "--bare", repoPath, destPath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: git clone --bare failed: %v\n", err)
				os.Exit(1)
			}
		}

	default:
		fmt.Fprintf(os.Stderr, "ERROR: path is not a git repository (no .git directory, no bare repo)\n")
		os.Exit(1)
	}

	fmt.Printf("Registered: %s/%s\n", owner, repo)
}

func detectRepoType(path string) string {
	if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
		return "working-tree"
	}
	for _, marker := range []string{"HEAD", "objects", "refs"} {
		if _, err := os.Stat(filepath.Join(path, marker)); os.IsNotExist(err) {
			return "none"
		}
	}
	return "bare"
}

func getRemoteURL(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

var (
	httpsRemote = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+?)(?:\.git)?$`)
	sshRemote   = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(?:\.git)?$`)
)

func parseGitHubURL(url string) (owner, repo string) {
	if m := httpsRemote.FindStringSubmatch(url); m != nil {
		return m[1], m[2]
	}
	if m := sshRemote.FindStringSubmatch(url); m != nil {
		return m[1], m[2]
	}
	return "", ""
}
