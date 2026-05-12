//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/respawn-app/ksrc/scripts/internal/depupdate"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/update-github-actions.go <workflow-dir>")
		os.Exit(2)
	}
	updates, err := depupdate.UpdateGitHubActions(os.Args[1], depupdate.GitHubActionsOptions{
		Token: os.Getenv("GITHUB_TOKEN"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, update := range updates {
		fmt.Println(update)
	}
}
