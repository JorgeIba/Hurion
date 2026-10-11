package workers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JorgeIba/Hurion/app/batchschema"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/infrastructure/storage"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/mock_services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

func TestScoreboardExporterWorker_ExportAll_WritesClassesJSON(t *testing.T) {
	tempDir := t.TempDir()
	store, err := storage.NewDiskStorage(tempDir)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	mockStore := mock_data.NewMockDataStore(ctrl)
	mockProblemSvc := mock_services.NewMockProblemService(ctrl)

	mockStore.EXPECT().GetAllClasses().Return([]data.Class{
		{ID: 1, ClassCode: "basicos", Name: "Algoritmos Básicos"},
	}, nil)

	mockStore.EXPECT().GetClassByClassCode("basicos").Return(data.Class{
		ID: 1, ClassCode: "basicos", Name: "Algoritmos Básicos",
	}, nil)

	mockStore.EXPECT().GetAllSolvedProblemsByClass(int64(1)).Return(map[data.User][]data.SolvedProblem{}, nil)
	mockProblemSvc.EXPECT().GetProblemsGroupedByTopic("basicos").Return([]service_model.ProblemsByTopic{}, nil)

	classSvc := services.NewClassService(mockStore)
	scoreSvc := &services.ScoreService{Datastore: mockStore, ProblemService: mockProblemSvc}

	worker := NewScoreboardExporterWorker(classSvc, scoreSvc, store)
	err = worker.ExportAll(context.Background())
	require.NoError(t, err)

	classesFile := tempDir + "/classes.json"
	assert.FileExists(t, classesFile)

	content, err := os.ReadFile(classesFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), `"classCode": "basicos"`)
	assert.Contains(t, string(content), `"name": "Algoritmos Básicos"`)

	for _, name := range []string{"ac-add-batch.json", "problem-add-batch.json", "problem-edit-batch.json"} {
		want, err := batchschema.FS.ReadFile(name)
		require.NoError(t, err)
		got, err := os.ReadFile(filepath.Join(tempDir, name))
		require.NoError(t, err)
		assert.Equal(t, want, got, "published schema %s must match the bot build", name)
	}

	scoreFile := tempDir + "/basicos_score.json"
	assert.FileExists(t, scoreFile)
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
