package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/respawn-app/ksrc/internal/gradle"
	"github.com/respawn-app/ksrc/internal/gradlehome"
	"github.com/respawn-app/ksrc/internal/resolve"
	"github.com/spf13/cobra"
)

func newDoctorCmd(app *App) *cobra.Command {
	var project string
	var userHome string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostics for ksrc",
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				project = "."
			}
			if _, err := app.Runner.LookPath("rg"); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "rg: not found on PATH\n")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "rg: ok\n")
			}

			wrapper := filepath.Join(project, "gradlew")
			if info, err := os.Stat(wrapper); err == nil && !info.IsDir() {
				fmt.Fprintf(cmd.OutOrStdout(), "gradle: ./gradlew\n")
			} else if _, err := app.Runner.LookPath("gradle"); err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "gradle: gradle on PATH\n")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "gradle: not found (no ./gradlew and gradle not on PATH)\n")
			}

			home, err := gradlehome.Resolve(userHome, project)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "gradle user home: error: %v\n", err)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "gradle user home: %s (%s)\n", home.Path, home.Source)
			}

			cache, err := resolve.GradleCacheDirWithOptions(resolve.CacheOptions{GradleUserHome: userHome, WorkDir: project})
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "gradle cache: error: %v\n", err)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "gradle cache: %s\n", cache)
			}

			_, err = gradle.Resolve(context.Background(), app.Runner, gradle.ResolveOptions{ProjectDir: project, GradleUserHome: userHome})
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "issue: gradle resolve failed: %v\n", err)
				fmt.Fprintf(cmd.OutOrStdout(), "suggestion: fix Gradle project (sync/build), then rerun ksrc\n")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "gradle resolve: ok\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", ".", "project root")
	cmd.Flags().StringVar(&userHome, "gradle-user-home", "", "Gradle user home (default: GRADLE_USER_HOME or ~/.gradle)")

	return cmd
}
