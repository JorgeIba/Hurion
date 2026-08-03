package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiskStorage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		baseDir     string
		expectedErr error
	}{
		{
			name:        "empty base directory",
			baseDir:     "",
			expectedErr: ErrEmptyBaseDir,
		},
		{
			name:        "whitespace base directory",
			baseDir:     "   ",
			expectedErr: ErrEmptyBaseDir,
		},
		{
			name:        "valid base directory",
			baseDir:     "/tmp/exports",
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store, err := NewDiskStorage(tt.baseDir)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.baseDir, store.BaseDir())
			}
		})
	}
}

func TestDiskStorage_Save(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		filename     string
		content      []byte
		expectedFile string
	}{
		{
			name:         "standard save success",
			filename:     "test_score.json",
			content:      []byte("hello world scoreboard"),
			expectedFile: "test_score.json",
		},
		{
			name:         "path sanitization removes slashes",
			filename:     "subfolder/test_score.json",
			content:      []byte("sanitized content"),
			expectedFile: "subfolder_test_score.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tempDir := t.TempDir()
			store, err := NewDiskStorage(tempDir)
			require.NoError(t, err)

			err = store.Save(context.Background(), tt.filename, tt.content)
			require.NoError(t, err)

			expectedPath := filepath.Join(tempDir, tt.expectedFile)
			data, err := os.ReadFile(expectedPath)
			require.NoError(t, err)
			assert.Equal(t, tt.content, data)
		})
	}
}
