package cmd

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kanafm/github-proxy/ca"
	"github.com/kanafm/github-proxy/proxy"
)

func Start(args []string) {
	flags := flag.NewFlagSet("start", flag.ExitOnError)
	port := flags.Int("port", 8443, "port to listen on")
	flags.Parse(args)

	dir := dataDirPath()

	caCrtPath := filepath.Join(dir, "ca.crt")
	if _, err := os.Stat(caCrtPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "ERROR: CA not found. Run 'github-proxy init' first.\n")
		os.Exit(1)
	}

	c, err := ca.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: loading CA: %v\n", err)
		os.Exit(1)
	}

	reposDir := filepath.Join(dir, "repos")
	if err := os.MkdirAll(reposDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: creating repos dir: %v\n", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", *port)

	srv := proxy.NewServer(addr, reposDir, c)

	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
