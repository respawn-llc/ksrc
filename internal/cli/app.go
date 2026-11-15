package cli

import "github.com/respawn-llc/ksrc/internal/executil"

type App struct {
	Runner  executil.Runner
	Verbose bool
}

func NewApp() *App {
	return &App{Runner: executil.OSRunner{}}
}
