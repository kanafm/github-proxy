package proxy

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kanafm/github-proxy/ca"
)

type Server struct {
	addr    string
	reposDir string
	ca      *ca.CA
	router  *Router
}

func NewServer(addr string, reposDir string, c *ca.CA) *Server {
	s := &Server{
		addr:    addr,
		reposDir: reposDir,
		ca:      c,
	}
	s.router = &Router{
		reposDir: reposDir,
		upstream: &Upstream{},
	}
	return s
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srv := &http.Server{
		Handler:      s,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		srv.Close()
	}()

	log.Printf("github-proxy listening on %s", s.addr)
	log.Printf("repos directory: %s", s.reposDir)

	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		s.handleConnect(w, r)
		return
	}

	resp, err := s.router.upstream.Forward(r, r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		return
	}

	targetHost := r.Host
	host, _, err := net.SplitHostPort(targetHost)
	if err != nil {
		host = targetHost
	}

	if host != "github.com" && host != "api.github.com" {
		s.tunnel(host, clientConn)
		return
	}

	mitm := &MITM{ca: s.ca, router: s.router}
	mitm.handle(clientConn, host)
}

func (s *Server) tunnel(host string, clientConn net.Conn) {
	log.Printf("TUNNEL: %s", host)
	upstreamConn, err := net.Dial("tcp", net.JoinHostPort(host, "443"))
	if err != nil {
		log.Printf("tunnel to %s failed: %v", host, err)
		return
	}
	defer upstreamConn.Close()

	go io.Copy(upstreamConn, clientConn)
	io.Copy(clientConn, upstreamConn)
}
