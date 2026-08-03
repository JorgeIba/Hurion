package workers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/JorgeIba/Hurion/app/infrastructure/storage"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	logger.Init()
}

func TestScoreboardExporterWorker_ExportAll(t *testing.T) {
	tempDir := t.TempDir()
	store, err := storage.NewDiskStorage(tempDir)
	require.NoError(t, err)

	// Test worker instantiation
	worker := NewScoreboardExporterWorker(&services.ClassService{}, &services.ScoreService{}, store)
	assert.NotNil(t, worker)

	// Verify ExportAll gracefully returns error when Datastore is not injected in services
	exportErr := worker.ExportAll(context.Background())
	assert.Error(t, exportErr)

	// Verify no files written when service fails
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestScoreboardExporterWorker_DebounceAndStop(t *testing.T) {
	// WHY THIS TEST MATTERS: Verifies that rapid Trigger() calls do not block or panic,
	// and that Stop() cleanly terminates the worker goroutine without deadlocking.
	tempDir := t.TempDir()
	store, err := storage.NewDiskStorage(tempDir)
	require.NoError(t, err)

	worker := NewScoreboardExporterWorker(&services.ClassService{}, &services.ScoreService{}, store)
	worker.DebounceDuration = 10 * time.Millisecond // Fast debounce for testing

	worker.Start()

	// Trigger multiple rapid events
	for i := 0; i < 5; i++ {
		worker.Trigger()
	}

	// Give worker time to process debounced event loop
	time.Sleep(30 * time.Millisecond)

	// Stop must drain channels and unblock without hanging
	worker.Stop()
}
