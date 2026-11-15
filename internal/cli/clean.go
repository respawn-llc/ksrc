package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/respawn-app/ksrc/internal/cache"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newCleanCmd() *cobra.Command {
	var dryRun, yes bool
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Preview and delete ksrc-owned caches",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			input, ok := cmd.InOrStdin().(*os.File)
			interactive := ok && term.IsTerminal(int(input.Fd()))
			return cleanCaches(cmd, dryRun, yes, interactive)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print cleanup summary without deleting (default in non-interactive environments)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "delete without confirmation")
	return cmd
}

func cleanCaches(cmd *cobra.Command, dryRun, yes, interactive bool) error {
	if interactive {
		fmt.Fprintln(cmd.ErrOrStderr(), "Scanning ksrc caches...")
	}
	roots, err := cache.Roots()
	if err != nil {
		return err
	}
	folders, err := cache.Inspect(cmd.Context(), roots)
	if err != nil {
		return err
	}
	if len(folders) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No ksrc caches to clean.")
		return err
	}
	out := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	var total int64
	for _, folder := range folders {
		fmt.Fprintf(out, "%s\t%s\n", folder.Path, formatCacheSize(folder.Bytes))
		total += folder.Bytes
	}
	fmt.Fprintf(out, "Total\t%s\n", formatCacheSize(total))
	if err := out.Flush(); err != nil {
		return err
	}
	if dryRun || (!interactive && !yes) {
		return nil
	}
	if !yes {
		fmt.Fprint(cmd.ErrOrStderr(), "Delete these cache folders? [y/N] ")
		answer, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
		if err != nil {
			return fmt.Errorf("read cleanup confirmation: %w", err)
		}
		answer = strings.TrimSpace(answer)
		if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
			return nil
		}
	}
	fmt.Fprintln(cmd.ErrOrStderr(), "Cleaning ksrc caches...")
	return cache.Delete(cmd.Context(), folders)
}

func formatCacheSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	size, suffix := float64(bytes), "B"
	for _, suffix = range []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"} {
		size /= unit
		if size < unit {
			break
		}
	}
	return fmt.Sprintf("%.1f %s", size, suffix)
}
