package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultDirPermissions sets directory perms: owner=rwx(7), group=r-x(5), others=r-x(5)
	DefaultDirPermissions os.FileMode = 0o755
	// DefaultFilePermissions sets file perms: owner=rw-(6), group=r--(4), others=r--(4)
	DefaultFilePermissions os.FileMode = 0o644
)

type DiskStorage struct {
	BaseDir string
}

func NewDiskStorage(baseDir string) *DiskStorage {
	return &DiskStorage{BaseDir: baseDir}
}

func (s *DiskStorage) Save(ctx context.Context, name string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context canceled before saving %s: %w", name, err)
	}

	cleanBaseDir := filepath.Clean(s.BaseDir)
	if err := os.MkdirAll(cleanBaseDir, DefaultDirPermissions); err != nil {
		return fmt.Errorf("failed to create base dir %s: %w", cleanBaseDir, err)
	}

	safeName := sanitizeFilename(name)
	fullPath := filepath.Join(cleanBaseDir, safeName)

	return writeAtomic(cleanBaseDir, fullPath, data, DefaultFilePermissions)
}

// sanitizeFilename replaces path separators and cleans dangerous characters
func sanitizeFilename(name string) string {
	cleaned := filepath.Base(filepath.Clean(name))
	cleaned = strings.ReplaceAll(cleaned, "/", "_")
	cleaned = strings.ReplaceAll(cleaned, "\\", "_")
	return cleaned
}

// writeAtomic writes bytes to a temp file in dir and atomically renames it over targetPath
func writeAtomic(dir string, targetPath string, data []byte, perm os.FileMode) error {
	tempFile, err := os.CreateTemp(dir, "export-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp export file: %w", err)
	}
	tempName := tempFile.Name()

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempName) // Cleanup temp file if rename didn't complete
	}()

	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("failed to write data to temp file: %w", err)
	}

	if err := tempFile.Chmod(perm); err != nil {
		return fmt.Errorf("failed to set temp file permissions: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tempName, targetPath); err != nil {
		return fmt.Errorf("failed to atomically rename temp file to %s: %w", targetPath, err)
	}

	return nil
}
