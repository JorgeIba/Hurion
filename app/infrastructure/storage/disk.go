package storage

import (
	"context"
	"errors"
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

var (
	ErrEmptyBaseDir  = errors.New("disk storage: base directory cannot be empty")
	ErrInvalidName   = errors.New("disk storage: invalid or empty file name")
	ErrPathTraversal = errors.New("disk storage: path traversal detected")
)

type DiskStorage struct {
	baseDir  string
	dirPerm  os.FileMode
	filePerm os.FileMode
}

// NewDiskStorage initializes a validated DiskStorage instance.
func NewDiskStorage(baseDir string) (*DiskStorage, error) {
	cleaned := filepath.Clean(strings.TrimSpace(baseDir))
	if cleaned == "" || cleaned == "." {
		return nil, ErrEmptyBaseDir
	}

	return &DiskStorage{
		baseDir:  cleaned,
		dirPerm:  DefaultDirPermissions,
		filePerm: DefaultFilePermissions,
	}, nil
}

func (s *DiskStorage) BaseDir() string {
	return s.baseDir
}

func (s *DiskStorage) Save(ctx context.Context, name string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context canceled before saving %s: %w", name, err)
	}

	safeName, err := sanitizeFilename(name)
	if err != nil {
		return fmt.Errorf("invalid filename %q: %w", name, err)
	}

	fullPath := filepath.Join(s.baseDir, safeName)
	if !isPathSafe(s.baseDir, fullPath) {
		return fmt.Errorf("%w: %s", ErrPathTraversal, name)
	}

	if err := os.MkdirAll(s.baseDir, s.dirPerm); err != nil {
		return fmt.Errorf("failed to create base dir %s: %w", s.baseDir, err)
	}

	return s.writeAtomic(ctx, fullPath, data)
}

func (s *DiskStorage) writeAtomic(ctx context.Context, targetPath string, data []byte) (err error) {
	tempFile, err := os.CreateTemp(s.baseDir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempName := tempFile.Name()

	defer func() {
		if tempFile != nil {
			_ = tempFile.Close()
		}
		if err != nil {
			_ = os.Remove(tempName)
		}
	}()

	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("failed to write data to temp file: %w", err)
	}

	if err := tempFile.Chmod(s.filePerm); err != nil {
		return fmt.Errorf("failed to set permissions on temp file: %w", err)
	}

	// Guarantee disk durability before renaming
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		tempFile = nil
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	tempFile = nil // Prevent deferred double-close

	if err := os.Rename(tempName, targetPath); err != nil {
		return fmt.Errorf("failed to atomically rename temp file to %s: %w", targetPath, err)
	}

	return nil
}

func sanitizeFilename(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", ErrInvalidName
	}

	replacer := strings.NewReplacer("/", "_", "\\", "_")
	sanitized := replacer.Replace(trimmed)
	cleaned := filepath.Base(filepath.Clean(sanitized))

	if cleaned == "." || cleaned == ".." || cleaned == "/" || cleaned == "\\" {
		return "", ErrInvalidName
	}

	return cleaned, nil
}

func isPathSafe(baseDir, targetPath string) bool {
	rel, err := filepath.Rel(baseDir, targetPath)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..") && rel != ".."
}
