package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func serveGitBackend(reposRootDir string, req *http.Request) (*http.Response, error) {
	log.Printf("GIT   %s", req.URL.String())
	cmd := exec.Command("git", "http-backend")
	cmd.Env = append(os.Environ(),
		"GIT_PROJECT_ROOT="+reposRootDir,
		"GIT_HTTP_EXPORT_ALL=1",
		"PATH_INFO="+req.URL.Path,
		"QUERY_STRING="+req.URL.RawQuery,
		"REQUEST_METHOD="+req.Method,
		"CONTENT_TYPE="+req.Header.Get("Content-Type"),
		"CONTENT_LENGTH="+strconv.FormatInt(req.ContentLength, 10),
	)

	if gp := req.Header.Get("Git-Protocol"); gp != "" {
		cmd.Env = append(cmd.Env, "GIT_PROTOCOL="+gp)
	}

	cmd.Stdin = req.Body

	stdout, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git http-backend: %w", err)
	}

	return parseCGIResponse(stdout)
}

func parseCGIResponse(data []byte) (*http.Response, error) {
	idx := bytes.Index(data, []byte("\r\n\r\n"))
	if idx == -1 {
		idx = bytes.Index(data, []byte("\n\n"))
		if idx == -1 {
			return &http.Response{
				StatusCode:    200,
				Status:        "200 OK",
				Proto:         "HTTP/1.1",
				ProtoMajor:    1,
				ProtoMinor:    1,
				ContentLength: int64(len(data)),
				Body:          io.NopCloser(bytes.NewReader(data)),
				Header:        make(http.Header),
			}, nil
		}
		idx += 2
	} else {
		idx += 4
	}

	headerData := data[:idx]
	body := data[idx:]

	header := make(http.Header)
	statusCode := 200
	statusText := "OK"

	for _, line := range strings.Split(string(headerData), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if colon := strings.Index(line, ":"); colon > 0 {
			key := strings.TrimSpace(line[:colon])
			val := strings.TrimSpace(line[colon+1:])
			if strings.EqualFold(key, "Status") {
				parts := strings.SplitN(val, " ", 2)
				if code, err := strconv.Atoi(parts[0]); err == nil {
					statusCode = code
				}
				if len(parts) > 1 {
					statusText = parts[1]
				}
			} else {
				header.Add(key, val)
			}
		}
	}

	return &http.Response{
		StatusCode:    statusCode,
		Status:        fmt.Sprintf("%d %s", statusCode, statusText),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(len(body)),
		Body:          io.NopCloser(bytes.NewReader(body)),
		Header:        header,
	}, nil
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	resp, err := r.Route(req, req.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
