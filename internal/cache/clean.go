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

func protectedDirectories() ([]string, error) {
	paths := []string{os.TempDir()}
	for _, get := range []func() (string, error){os.UserHomeDir, os.UserCacheDir} {
		path, err := get()
		if err == nil {
			paths = append(paths, path)
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	paths = append(paths, cwd)
	var resolved []string
	for _, path := range paths {
		path, err := filepath.EvalSymlinks(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, path)
	}
	return resolved, nil
}

// Inspect previews resolved locations. Root symlinks are unlinked rather than
// followed, and sizes count regular-file bytes rather than physical disk blocks.
func Inspect(ctx context.Context, roots []string) ([]Folder, error) {
	protected, err := protectedDirectories()
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, path := range roots {
		path, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		parent, err := filepath.EvalSymlinks(filepath.Dir(path))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("resolve cache %q: %w", path, err)
		}
		path = filepath.Join(parent, filepath.Base(path))
		if filepath.Dir(path) == path {
			return nil, fmt.Errorf("refusing to clean filesystem root %q", path)
		}
		for _, shared := range protected {
			if contains(path, shared) {
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
		err = filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
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
