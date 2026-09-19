// Command ark is the official AI-Ark command-line interface.
//
// It is a thin, stateless client over the AI-Ark developer-portal REST API:
// every request is authenticated with the caller's own API token and billed
// exactly as any other API call. The CLI holds no server-side state of its own.
package main

import (
	"os"

	"github.com/ai-ark-com/ark-cli/cmd"
)

// Version metadata, overridden at build time via -ldflags (see .goreleaser.yaml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(cmd.Execute(cmd.BuildInfo{Version: version, Commit: commit, Date: date}))
}
