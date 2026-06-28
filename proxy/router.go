package proxy

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Router struct {
	reposDir string
	upstream *Upstream
}

func (r *Router) Route(req *http.Request, host string) (*http.Response, error) {
	owner, repo, ok := parseRepo(req.URL.Path)
	if !ok {
		return r.upstream.Forward(req, host)
	}

	repoName := strings.TrimSuffix(repo, ".git")
	localPath := filepath.Join(r.reposDir, owner, repoName+".git")
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return r.upstream.Forward(req, host)
	}

	path := req.URL.Path

	switch {
	case isGitSmartHTTP(path):
		return serveGitBackend(r.reposDir, req)
	case isAPICommits(path):
		return serveAPI(localPath, req)
	case isArchive(path):
		return serveArchive(localPath, req)
	default:
		return r.upstream.Forward(req, host)
	}
}

func parseRepo(path string) (owner, repo string, ok bool) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 2 {
		return "", "", false
	}

	if parts[0] == "repos" {
		if len(parts) < 4 {
			return "", "", false
		}
		return parts[1], parts[2], true
	}

	return parts[0], parts[1], true
}

func isGitSmartHTTP(path string) bool {
	return strings.HasSuffix(path, "/git-upload-pack") ||
		strings.HasSuffix(path, "/git-receive-pack") ||
		strings.Contains(path, "/info/refs")
}

func isAPICommits(path string) bool {
	return strings.Contains(path, "/repos/") && strings.Contains(path, "/commits/")
}

func isArchive(path string) bool {
	return strings.Contains(path, "/archive/") || strings.Contains(path, "/tarball/")
}
