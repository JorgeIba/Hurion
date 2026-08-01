package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	if err := os.MkdirAll(s.BaseDir, DefaultDirPermissions); err != nil {
		return fmt.Errorf("failed to create base dir %s: %w", s.BaseDir, err)
	}

	fullPath := filepath.Join(s.BaseDir, name)
	if err := os.WriteFile(fullPath, data, DefaultFilePermissions); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}
