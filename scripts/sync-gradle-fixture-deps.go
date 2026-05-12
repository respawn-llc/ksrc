//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/respawn-app/ksrc/scripts/internal/depupdate"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/sync-gradle-fixture-deps.go <libs.versions.toml> <fixture-build.gradle>")
		os.Exit(2)
	}
	if err := depupdate.SyncGradleFixtureDeps(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
