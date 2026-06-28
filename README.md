# github-proxy

This is a tiny local MITM HTTPS proxy that intercepts requests to `github.com` and `api.github.com`, serving registered repos from a local directory instead. Unregistered repos pass through transparently to real GitHub.

No changes should be required to `flake.nix`, `go.mod`, or Nix derivations as the proxy is transparent to all consumers (Nix flakes, Go modules, `git clone`).

Developing Nix flakes with `github:` inputs normally requires pushing to GitHub first. Go modules with `require github.com/...` need tags published on the remote. This proxy lets you:

- Iterate on a Nix flake library and its consumer locally without pushing
- Develop Go modules and their dependents without publishing tags
- Test full integration pipelines before any code leaves your machine

## Build

```bash
nix build
```

No external Go dependencies, it's a pure standard library.

## Quick Start

```bash
# 1. Build
nix build

# 2. Initialize (generates CA, prints setup instructions)
./result/bin/github-proxy init

# 3. Follow the printed instructions to:
#    - Trust the CA in macOS keychain
#    - Configure Nix daemon (if using Nix)
#    - Add proxy env vars to your shell
```

**Don't** set these permanently in `~/.zshrc` — they break everything when the proxy is off. Instead, add toggle functions to `~/.zshrc`:

```bash
github-proxy-start() {
  github-proxy start --port 8443 &
  eval $(github-proxy env --port=8443)
  echo "Proxy on (PID $!). Run github-proxy-stop to disable."
}

github-proxy-stop() {
  pkill -f 'github-proxy start' 2>/dev/null
  eval $(github-proxy env --unset)
  echo "Proxy off."
}
```

Finally,
```bash
github-proxy-start

# 4. Register a local repo
./result/bin/github-proxy add ~/projects/my-library

# 5. Start the proxy
./result/bin/github-proxy start

# 6. Everything just works
nix build       # github: inputs resolve from local repos
go build        # go.mod deps resolve from local repos
git clone https://github.com/me/mylib  # clones from local if registered

github-proxy-stop
```

## *Deep dive*

<details>
<summary>Learn more...</summary>

## 1. Commands

### `init`

Generates an ECDSA P-256 CA certificate in `~/.github-proxy/`, creates a combined Nix CA bundle, and prints setup instructions for macOS, Nix, and shell configuration.

```
github-proxy init
```

### `start`

Starts the MITM proxy server. The proxy listens on `127.0.0.1` only (no LAN exposure).

```
github-proxy start [--port <port>]
```

Default port: `8443`

### `add`

Registers a local git repository. The proxy auto-detects `owner/repo` from the git remote URL. You can also specify them manually.

```
github-proxy add <path> [--owner <owner>] [--repo <name>]
```

Examples:
```
github-proxy add ~/projects/my-library                     # auto-detect from remote
github-proxy add ~/projects/my-lib --owner me --repo mylib  # manual
```

The command creates a bare clone at `~/.github-proxy/repos/<owner>/<repo>.git`.

### `list`

Lists all registered repos.

```
github-proxy list
```

### `remove`

Removes a registered repo.

```
github-proxy remove <owner>/<repo>
```

Example:
```
github-proxy remove me/mylib
```

## 2. Behind the scenes

1. `nix build` / `go build` / `git clone`
2. `CONNECT github.com:443`
3. `MITM Proxy :8443` with TLS terminate (CA), parses owner/repo, and checks local repos.
4. If local repo exists, serve that.
5. If not, forward to real GitHub.

Data is stored in `~/.github-proxy` where `repos` contain symlinks to your local repos when you use the `add` command:

```
~/.github-proxy/
├── ca.key              # CA private key (0600)
├── ca.crt              # CA certificate (0644)
└── repos/
    └── owner/
        └── repo.git/   # bare clone or symlink
```


## 3. Nix Daemon Configuration for macOS

The proxy instructions printed by `init` include the commands to configure the Nix daemon via `PlistBuddy`. After running them, reload the daemon:

```bash
sudo launchctl unload /Library/LaunchDaemons/org.nixos.nix-daemon.plist
sudo launchctl load /Library/LaunchDaemons/org.nixos.nix-daemon.plist
```

Then add the combined CA bundle to the sandbox:

```bash
echo 'extra-sandbox-paths = /etc/nix/ca-bundle.crt' | sudo tee -a /etc/nix/nix.conf
```

## 4. Typical usage

bloggo is a Go library at `github.com/kanafm/bloggo`. bloggo-kana is a consumer that depends on it via both `go.mod` and `flake.nix`.

### Setup

```bash
# Initialize git in both repos (one-time)
git -C ~/projects/bloggo init && git -C ~/projects/bloggo add -A && git -C ~/projects/bloggo commit -m "init"
git -C ~/projects/bloggo remote add origin https://github.com/kanafm/bloggo
git -C ~/projects/bloggo tag v0.1.0

# Same for bloggo-kana
git -C ~/projects/bloggo-kana init && git -C ~/projects/bloggo-kana add -A && git -C ~/projects/bloggo-kana commit -m "init"
git -C ~/projects/bloggo-kana remote add origin https://github.com/kanafm/bloggo-kana

# Register bloggo with the proxy
github-proxy add ~/projects/bloggo

# Verify
github-proxy list
# → kanafm/bloggo
```

### Daily workflow

```bash
# Start the proxy + set env vars
github-proxy-start

# Go build (uses proxy for module resolution)
cd ~/projects/bloggo-kana
go build ./...
./bloggo-kana
# → Hello, World!

# Stop when done
github-proxy-stop
```

### Turn it off

```bash
# Ctrl+C the proxy, or:
kill $(pgrep github-proxy)

# Now everything hits real GitHub again
# (bloggo doesn't exist on GitHub → builds will fail — as expected)
```


## 5. Gotchas

- **CA staleness**: The combined bundle copies Nix's cert bundle at `init` time. Re-run `init` and re-copy to `/etc/nix/ca-bundle.crt` if Nix's cacert package updates.
- **Bloggo not pushed**: If a repo only exists locally, builds fail when the proxy is off. Push to GitHub before turning the proxy off.
- **Go checksum DB**: Set `GONOSUMDB` for private modules. For proxy.golang.org, set `GOPROXY=direct` or the proxy will need to trust our CA for that domain too.
- **macOS firewall**: Binding to `127.0.0.1` only avoids local network permission dialogs.
- **Port conflicts**: 8443 is the default. Use `--port` if it's taken.

</details>
