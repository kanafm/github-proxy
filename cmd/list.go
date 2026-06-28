package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

func List() {
	dir := dataDirPath()
	reposDir := filepath.Join(dir, "repos")

	entries, err := os.ReadDir(reposDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: reading repos dir: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("No repos registered.")
		return
	}

	for _, ownerEntry := range entries {
		if !ownerEntry.IsDir() {
			continue
		}
		owner := ownerEntry.Name()
		ownerDir := filepath.Join(reposDir, owner)
		repoEntries, err := os.ReadDir(ownerDir)
		if err != nil {
			continue
		}
		for _, repoEntry := range repoEntries {
			if !repoEntry.IsDir() {
				continue
			}
			repo := repoEntry.Name()
			repo = repo[:len(repo)-len(".git")]
			fmt.Printf("%s/%s\n", owner, repo)
		}
	}
}
