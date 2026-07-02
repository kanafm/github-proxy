package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/kanafm/github-proxy/cmd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: github-proxy <command> [args...]")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  init      Generate CA certificate and print setup instructions")
		fmt.Println("  start     Start the proxy server")
		fmt.Println("  env       Print environment variables to source")
		fmt.Println("  add       Register a local repo")
		fmt.Println("  list      List registered repos")
		fmt.Println("  remove    Remove a registered repo")
		fmt.Println("  update    Fetch new branches/tags into a registered repo")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		cmd.Init()
	case "start":
		cmd.Start(os.Args[2:])
	case "add":
		cmd.Add(os.Args[2:])
	case "list":
		cmd.List()
	case "remove":
		cmd.Remove(os.Args[2:])
	case "update":
		cmd.Update(os.Args[2:])
	case "env":
		unset := false
		port := "8443"
		for _, a := range os.Args[2:] {
			if a == "--unset" {
				unset = true
			}
			if strings.HasPrefix(a, "--port=") {
				port = strings.TrimPrefix(a, "--port=")
			}
		}
		cmd.Env(unset, port)
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
