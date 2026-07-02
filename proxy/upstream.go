package proxy

import (
	"crypto/tls"
	"io"
	"log"
	"net/http"
)

type Upstream struct{}

func (u *Upstream) Forward(req *http.Request, targetHost string) (*http.Response, error) {
	log.Printf("FWD   %s %s → %s", req.Method, req.URL.String(), targetHost)
	outReq := cloneRequest(req)
	outReq.URL.Scheme = "https"
	outReq.URL.Host = targetHost
	outReq.RequestURI = ""

	outReq.Header.Del("Proxy-Connection")
	outReq.Header.Del("Proxy-Authorization")

	removeHopByHopHeaders(outReq.Header)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			ServerName: targetHost,
		},
		ForceAttemptHTTP2: false,
	}

	return transport.RoundTrip(outReq)
}

func cloneRequest(req *http.Request) *http.Request {
	outReq := req.Clone(req.Context())

	outReq.Header = make(http.Header)
	for k, v := range req.Header {
		outReq.Header[k] = v
	}

	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			outReq.Body = io.NopCloser(io.LimitReader(
				&byteReader{data: bodyBytes}, int64(len(bodyBytes)),
			))
			outReq.ContentLength = int64(len(bodyBytes))
			outReq.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(
					&byteReader{data: bodyBytes},
				), nil
			}
		}
	}

	return outReq
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func removeHopByHopHeaders(header http.Header) {
	hopByHop := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	}
	for _, h := range hopByHop {
		header.Del(h)
	}
	if header.Get("Connection") != "" {
		header.Del("Connection")
	}
}
