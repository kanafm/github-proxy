package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kanafm/github-proxy/ca"
)

const dataDir = ".github-proxy"

func dataDirPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, dataDir)
}

func Init() {
	dir := dataDirPath()

	if _, err := os.Stat(dir); err == nil {
		fmt.Printf("WARNING: %s already exists. Overwriting ca.key and ca.crt.\n", dir)
	}

	c, err := ca.Generate(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: generating CA: %v\n", err)
		os.Exit(1)
	}
	_ = c

	home, _ := os.UserHomeDir()
	reposDir := filepath.Join(dir, "repos")
	if err := os.MkdirAll(reposDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: creating repos directory: %v\n", err)
		os.Exit(1)
	}

	bundleSrc := "/nix/var/nix/profiles/default/etc/ssl/certs/ca-bundle.crt"
	caCrtPath := filepath.Join(dir, "ca.crt")
	bundleDst := filepath.Join(home, "nix-ca-bundle.crt")

	nixBundle, err := os.ReadFile(bundleSrc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: could not read Nix CA bundle at %s: %v\n", bundleSrc, err)
		fmt.Printf("Skipping combined CA bundle creation.\n")
	} else {
		caCrt, err := os.ReadFile(caCrtPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: could not read %s: %v\n", caCrtPath, err)
		} else {
			combined := append(nixBundle, caCrt...)
			if err := os.WriteFile(bundleDst, combined, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "WARNING: could not write combined bundle: %v\n", err)
			} else {
				fmt.Printf("Combined CA bundle written to %s\n", bundleDst)
			}
		}
	}

	fmt.Println()
	fmt.Println("=== One-time setup (copy-paste this entire block) ===")
	fmt.Printf("sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s && \\\n", caCrtPath)
	fmt.Printf("  sudo cp %s /etc/nix/ca-bundle.crt && \\\n", bundleDst)
	fmt.Println("  sudo sed -i '' '/^extra-sandbox-paths/d' /etc/nix/nix.conf && \\")
	fmt.Println("  echo 'extra-sandbox-paths = /etc/nix/ca-bundle.crt' | sudo tee -a /etc/nix/nix.conf && \\")
	fmt.Println("  (sudo /usr/libexec/PlistBuddy -c \"Print :EnvironmentVariables\" /Library/LaunchDaemons/org.nixos.nix-daemon.plist 2>/dev/null || \\")
	fmt.Println("   sudo /usr/libexec/PlistBuddy -c \"Add :EnvironmentVariables dict\" /Library/LaunchDaemons/org.nixos.nix-daemon.plist) && \\")
	fmt.Println("  sudo /usr/libexec/PlistBuddy -c \"Set :EnvironmentVariables:https_proxy http://localhost:8443\" /Library/LaunchDaemons/org.nixos.nix-daemon.plist && \\")
	fmt.Println(`  sudo /usr/libexec/PlistBuddy -c "Set :EnvironmentVariables:NIX_SSL_CERT_FILE /etc/nix/ca-bundle.crt" /Library/LaunchDaemons/org.nixos.nix-daemon.plist && \`)
	fmt.Println("  sudo launchctl unload /Library/LaunchDaemons/org.nixos.nix-daemon.plist && \\")
	fmt.Println("  sudo launchctl load /Library/LaunchDaemons/org.nixos.nix-daemon.plist")
	fmt.Println()

	fmt.Println("=== Shell: Add to ~/.zshrc ===")
	fmt.Println("# Run 'github-proxy-start' to enable, 'github-proxy-stop' to disable.")
	fmt.Println("# Optionally pass a port: github-proxy-start 9999")
	fmt.Println("github-proxy-start() {")
	fmt.Println("  local port=${1:-8443}")
	fmt.Println("  github-proxy start --port $port &")
	fmt.Println("  eval $(github-proxy env --port=$port)")
	fmt.Println("  echo 'Proxy on (PID' $! '). Run github-proxy-stop to disable.'")
	fmt.Println("}")
	fmt.Println()
	fmt.Println("github-proxy-stop() {")
	fmt.Println("  pkill -f 'github-proxy start' 2>/dev/null")
	fmt.Println("  eval $(github-proxy env --unset)")
	fmt.Println("  echo 'Proxy off.'")
	fmt.Println("}")
	fmt.Println()
	fmt.Println("=== Go: Skip checksum DB for private modules ===")
	fmt.Println("export GONOSUMDB=github.com/kanafm/*")
}
