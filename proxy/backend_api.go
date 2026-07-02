package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

func serveAPI(repoPath string, req *http.Request) (*http.Response, error) {
	log.Printf("API   %s", req.URL.String())
	ref := extractRef(req.URL.Path)

	sha, err := gitCmd(repoPath, "rev-parse", ref)
	if err != nil {
		log.Printf("git rev-parse %s failed: %v, falling through to upstream", ref, err)
		return nil, fmt.Errorf("ref not found: %s", ref)
	}

	tree, err := gitCmd(repoPath, "rev-parse", ref+"^{tree}")
	if err != nil {
		log.Printf("git rev-parse %s^{tree} failed: %v", ref, err)
		return nil, fmt.Errorf("tree not found for: %s", ref)
	}

	json := fmt.Sprintf(`{"sha":"%s","commit":{"tree":{"sha":"%s"}}}`, sha, tree)

	return &http.Response{
		StatusCode:    200,
		Status:        "200 OK",
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(len(json)),
		Body:          io.NopCloser(strings.NewReader(json)),
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}, nil
}

func extractRef(path string) string {
	idx := strings.Index(path, "/commits/")
	if idx == -1 {
		return "HEAD"
	}
	return path[idx+len("/commits/"):]
}

func gitCmd(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
