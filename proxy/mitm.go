package proxy

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
	"net/http"

	"github.com/kanafm/github-proxy/ca"
)

type MITM struct {
	ca     *ca.CA
	router *Router
}

func (m *MITM) handle(clientConn net.Conn, host string) {
	log.Printf("MITM: %s", host)
	tlsConfig := &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			serverName := hello.ServerName
			if serverName == "" {
				serverName = host
			}
			return m.ca.SignHost(serverName)
		},
	}

	tlsConn := tls.Server(clientConn, tlsConfig)
	defer tlsConn.Close()

	reader := bufio.NewReader(tlsConn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			break
		}

		resp, err := m.router.Route(req, host)
		if err != nil {
			log.Printf("MITM %s ERROR: %v", req.URL.Path, err)
			resp = errorResponse(err)
		} else {
			log.Printf("MITM %s → %d", req.URL.Path, resp.StatusCode)
		}

		if err := resp.Write(tlsConn); err != nil {
			req.Body.Close()
			break
		}

		req.Body.Close()
		resp.Body.Close()
	}
}

func errorResponse(err error) *http.Response {
	return &http.Response{
		StatusCode: http.StatusInternalServerError,
		Status:     http.StatusText(http.StatusInternalServerError),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body:       http.NoBody,
		Header: http.Header{
			"Content-Type": []string{"text/plain"},
		},
	}
}
