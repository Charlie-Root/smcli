package main

import (
	"os"

	"github.com/Charlie-Root/smcli/cmd"
)

// version is set at build time by goreleaser (-X main.version).
var version = "dev"

func main() {
	os.Exit(cmd.Execute(version))
}
