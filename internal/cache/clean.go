package cache

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type Folder struct {
	Path  string
	Bytes int64
}

func contains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && filepath.IsLocal(rel)
}

// Inspect never follows symlinks. Sizes are logical file bytes, not a promise
// about physical disk blocks reclaimed.
func Inspect(ctx context.Context, roots []string) ([]Folder, error) {
	protected := []string{os.TempDir()}
	for _, get := range []func() (string, error){os.UserHomeDir, os.UserCacheDir, os.Getwd} {
		path, err := get()
		if err != nil {
			return nil, err
		}
		protected = append(protected, path)
	}
	var paths []string
	for _, root := range roots {
		path, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		if filepath.Dir(path) == path {
			return nil, fmt.Errorf("refusing to clean filesystem root %q", path)
		}
		for _, protectedPath := range protected {
			if contains(path, protectedPath) {
				return nil, fmt.Errorf("refusing to clean shared directory %q", path)
			}
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var folders []Folder
	for _, path := range paths {
		covered := false
		for _, folder := range folders {
			if contains(folder.Path, path) {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		info, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect cache %q: %w", path, err)
		}
		if !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			return nil, fmt.Errorf("cache path %q is not a folder", path)
		}
		folder := Folder{Path: path}
		err = filepath.WalkDir(path, func(_ string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry.Type().IsRegular() {
				info, err := entry.Info()
				if err != nil {
					return err
				}
				folder.Bytes += info.Size()
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("measure cache %q: %w", path, err)
		}
		folders = append(folders, folder)
	}
	return folders, nil
}

func Delete(ctx context.Context, folders []Folder) error {
	var failures []error
	for _, folder := range folders {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if err := os.RemoveAll(folder.Path); err != nil {
			failures = append(failures, fmt.Errorf("clean cache %q: %w", folder.Path, err))
		}
	}
	return errors.Join(failures...)
}
