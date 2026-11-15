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
	Path   string
	Bytes  int64
	target string
}

func contains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && filepath.IsLocal(rel)
}

// canonicalPath resolves existing ancestors, including for paths not yet created.
func canonicalPath(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if errors.Is(err, fs.ErrNotExist) && filepath.Dir(path) != path {
		parent, err := canonicalPath(filepath.Dir(path))
		if err != nil {
			return "", err
		}
		return filepath.Join(parent, filepath.Base(path)), nil
	}
	return resolved, err
}

// Resolve the parent only: a symlink at the cache root itself is unlinked,
// not traversed.
func cacheTarget(path string) (string, error) {
	parent, err := canonicalPath(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

// Inspect does not follow links within cache folders. Sizes are logical file
// bytes, not a promise about physical disk blocks reclaimed.
func Inspect(ctx context.Context, roots []string) ([]Folder, error) {
	protected := []string{os.TempDir()}
	for _, get := range []func() (string, error){os.UserHomeDir, os.UserCacheDir} {
		path, err := get()
		if err == nil {
			protected = append(protected, path)
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	protected = append(protected, cwd)
	for i, path := range protected {
		protected[i], err = canonicalPath(path)
		if err != nil {
			return nil, err
		}
	}
	var paths []Folder
	for _, root := range roots {
		path, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		target, err := cacheTarget(path)
		if err != nil {
			return nil, fmt.Errorf("resolve cache %q: %w", path, err)
		}
		if filepath.Dir(target) == target {
			return nil, fmt.Errorf("refusing to clean filesystem root %q", path)
		}
		for _, protectedPath := range protected {
			if contains(target, protectedPath) {
				return nil, fmt.Errorf("refusing to clean shared directory %q", path)
			}
		}
		paths = append(paths, Folder{Path: path, target: target})
	}
	sort.Slice(paths, func(i, j int) bool { return paths[i].target < paths[j].target })
	var folders []Folder
	for _, folder := range paths {
		path, target := folder.Path, folder.target
		covered := false
		for _, existing := range folders {
			if contains(existing.target, target) {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		info, err := os.Lstat(target)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect cache %q: %w", path, err)
		}
		if !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			return nil, fmt.Errorf("cache path %q is not a folder", path)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			folders = append(folders, folder)
			continue
		}
		err = measure(ctx, &folder, info)
		if err != nil {
			return nil, fmt.Errorf("measure cache %q: %w", path, err)
		}
		folders = append(folders, folder)
	}
	return folders, nil
}

func measure(ctx context.Context, folder *Folder, info fs.FileInfo) error {
	root, err := os.OpenRoot(folder.target)
	if err != nil {
		return err
	}
	defer root.Close()
	opened, err := fs.Stat(root.FS(), ".")
	if err != nil {
		return err
	}
	if !os.SameFile(info, opened) {
		return fmt.Errorf("cache folder changed during inspection")
	}
	return fs.WalkDir(root.FS(), ".", func(_ string, entry fs.DirEntry, walkErr error) error {
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
}

func Delete(ctx context.Context, folders []Folder) error {
	var failures []error
	for _, folder := range folders {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if err := deleteFolder(folder); err != nil {
			failures = append(failures, fmt.Errorf("clean cache %q: %w", folder.Path, err))
		}
	}
	return errors.Join(failures...)
}

func deleteFolder(folder Folder) error {
	target, err := cacheTarget(folder.Path)
	if err != nil {
		return err
	}
	if target != folder.target {
		return fmt.Errorf("cache location changed since preview; run clean again")
	}
	parent, err := os.OpenRoot(filepath.Dir(target))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer parent.Close()
	return parent.RemoveAll(filepath.Base(target))
}
