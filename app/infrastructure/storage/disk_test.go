package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiskStorage_Save(t *testing.T) {
	tempDir := t.TempDir()
	store := NewDiskStorage(tempDir)

	content := []byte("hello world scoreboard")
	filename := "test_score.json"

	err := store.Save(context.Background(), filename, content)
	require.NoError(t, err)

	expectedPath := filepath.Join(tempDir, filename)
	data, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}
