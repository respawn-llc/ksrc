package fileidcache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	cacheDirEnv   = "KSRC_FILEID_CACHE_DIR"
	cacheRootName = "ksrc/file-id-locations-v1"
)

type entry struct {
	FileID  string `json:"fileId"`
	JarPath string `json:"jarPath"`
}

func Register(fileID string, jarPath string) error {
	fileID = strings.TrimSpace(fileID)
	jarPath = strings.TrimSpace(jarPath)
	if fileID == "" || jarPath == "" {
		return nil
	}
	root, err := cacheRoot()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	path := cachePath(root, fileID)
	encoded, err := json.Marshal(entry{FileID: fileID, JarPath: jarPath})
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(root, filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()
	if _, err := temp.Write(encoded); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func Lookup(fileID string) (string, bool, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return "", false, nil
	}
	root, err := cacheRoot()
	if err != nil {
		return "", false, err
	}
	encoded, err := os.ReadFile(cachePath(root, fileID))
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	var stored entry
	if err := json.Unmarshal(encoded, &stored); err != nil {
		return "", false, nil
	}
	if stored.FileID != fileID || strings.TrimSpace(stored.JarPath) == "" {
		return "", false, nil
	}
	if _, err := os.Stat(stored.JarPath); err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return stored.JarPath, true, nil
}

func cacheRoot() (string, error) {
	if path := strings.TrimSpace(os.Getenv(cacheDirEnv)); path != "" {
		return path, nil
	}
	root, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve file-id cache dir: %w", err)
	}
	return filepath.Join(root, cacheRootName), nil
}

func cachePath(root string, fileID string) string {
	sum := sha256.Sum256([]byte(fileID))
	return filepath.Join(root, hex.EncodeToString(sum[:])+".json")
}
