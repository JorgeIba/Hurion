package workers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/JorgeIba/Hurion/app/infrastructure/storage"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/JorgeIba/Hurion/app/presentation/jsonexport"
	"github.com/JorgeIba/Hurion/app/services"
)

const (
	DefaultDebounceDuration = 800 * time.Millisecond
	DefaultExportTimeout    = 10 * time.Second
)

type ScoreboardExporterWorker struct {
	ClassService       *services.ClassService
	ScoreService       *services.ScoreService
	Storage            storage.BlobStorage
	DebounceDuration   time.Duration
	updateEventChannel chan struct{}
	shutdownSignal     chan struct{}
	workerWaitGroup    sync.WaitGroup
}

func NewScoreboardExporterWorker(classSvc *services.ClassService, scoreSvc *services.ScoreService, store storage.BlobStorage) *ScoreboardExporterWorker {
	return &ScoreboardExporterWorker{
		ClassService:       classSvc,
		ScoreService:       scoreSvc,
		Storage:            store,
		DebounceDuration:   DefaultDebounceDuration,
		updateEventChannel: make(chan struct{}, 1),
		shutdownSignal:     make(chan struct{}),
	}
}

func (w *ScoreboardExporterWorker) Start() {
	w.workerWaitGroup.Add(1)
	go func() {
		defer w.workerWaitGroup.Done()
		for {
			select {
			case <-w.updateEventChannel:
				// Debounce rapid consecutive database update notifications
				select {
				case <-w.shutdownSignal:
					return
				case <-time.After(w.DebounceDuration):
				}

				// Drain any extra event triggers accumulated during debounce delay
				select {
				case <-w.updateEventChannel:
				default:
				}

				ctx, cancel := context.WithTimeout(context.Background(), DefaultExportTimeout)
				if err := w.ExportAll(ctx); err != nil {
					if logger.Log != nil {
						logger.Log.Errorf("error executing background scoreboard export: %v", err)
					}
				}
				cancel()
			case <-w.shutdownSignal:
				return
			}
		}
	}()
}

func (w *ScoreboardExporterWorker) Trigger() {
	select {
	case w.updateEventChannel <- struct{}{}:
	default:
		// Channel is full; an export run is already pending
	}
}

func (w *ScoreboardExporterWorker) Stop() {
	close(w.shutdownSignal)
	w.workerWaitGroup.Wait()
}

func (w *ScoreboardExporterWorker) ExportAll(ctx context.Context) error {
	if w.ClassService == nil || w.ClassService.Datastore == nil {
		return fmt.Errorf("class service or datastore is uninitialized")
	}

	classes, err := w.ClassService.GetAllClasses()
	if err != nil {
		return fmt.Errorf("failed to get classes: %w", err)
	}

	for _, c := range classes {
		score, err := w.ScoreService.GetScoreByClassCode(c.ClassCode)
		if err != nil {
			logger.Log.Errorf("failed to get score for class %s: %v", c.ClassCode, err)
			continue
		}

		jsonBytes, err := jsonexport.FormatScoreboard(c.ClassCode, c.Name, score)
		if err != nil {
			logger.Log.Errorf("failed to format scoreboard JSON for class %s: %v", c.ClassCode, err)
			continue
		}

		filename := fmt.Sprintf("%s_score.json", c.ClassCode)
		if err := w.Storage.Save(ctx, filename, jsonBytes); err != nil {
			logger.Log.Errorf("failed to save scoreboard for class %s: %v", c.ClassCode, err)
		}
	}

	return nil
}
