package cmd

import (
	"fmt"
)

func Env(unset bool, port string) {

	vars := []struct{ k, v string }{
		{"http_proxy", fmt.Sprintf("http://localhost:%s", port)},
		{"HTTP_PROXY", fmt.Sprintf("http://localhost:%s", port)},
		{"https_proxy", fmt.Sprintf("http://localhost:%s", port)},
		{"HTTPS_PROXY", fmt.Sprintf("http://localhost:%s", port)},
		{"GIT_SSL_CAINFO", "/etc/nix/ca-bundle.crt"},
		{"NIX_SSL_CERT_FILE", "/etc/nix/ca-bundle.crt"},
		{"GONOSUMDB", "github.com/kanafm/*"},
		{"GOPROXY", "direct"},
	}

	for _, v := range vars {
		if unset {
			fmt.Printf("unset %s\n", v.k)
		} else {
			fmt.Printf("export %s=%s\n", v.k, v.v)
		}
	}
}
