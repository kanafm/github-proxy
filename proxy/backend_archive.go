package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

func serveArchive(repoPath string, req *http.Request) (*http.Response, error) {
	log.Printf("ARCH  %s", req.URL.String())
	sha := extractArchiveSHA(req.URL.Path)
	if sha == "" {
		return nil, fmt.Errorf("no SHA in archive path: %s", req.URL.Path)
	}

	verify := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", sha+"^{commit}")
	if err := verify.Run(); err != nil {
		log.Printf("git rev-parse --verify %s failed: %v, falling through to upstream", sha, err)
		return nil, fmt.Errorf("commit not found: %s", sha)
	}

	cmd := exec.Command("git", "-C", repoPath, "archive", "--format=tar.gz", sha)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git archive: %w (stderr: %s)", err, stderr.String())
	}

	return &http.Response{
		StatusCode:    200,
		Status:        "200 OK",
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(stdout.Len()),
		Body:          io.NopCloser(bytes.NewReader(stdout.Bytes())),
		Header: http.Header{
			"Content-Type": []string{"application/x-gzip"},
		},
	}, nil
}

func extractArchiveSHA(path string) string {
	if idx := strings.Index(path, "/archive/"); idx != -1 {
		rest := path[idx+len("/archive/"):]
		sha := strings.TrimSuffix(rest, ".tar.gz")
		sha = strings.TrimSuffix(sha, "/")
		return sha
	}
	if idx := strings.Index(path, "/tarball/"); idx != -1 {
		rest := path[idx+len("/tarball/"):]
		sha := strings.TrimSuffix(rest, "/")
		return sha
	}
	return ""
}
