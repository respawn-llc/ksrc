package cli

import "github.com/spf13/cobra"

var followupResolutionFlags = []string{
	"project",
	"module",
	"group",
	"artifact",
	"version",
	"scope",
	"config",
	"targets",
	"subproject",
	"offline",
	"refresh",
	"buildsrc",
	"buildscript",
	"include-builds",
}

func hasExplicitFollowupResolutionContext(cmd *cobra.Command) bool {
	for _, name := range followupResolutionFlags {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}
