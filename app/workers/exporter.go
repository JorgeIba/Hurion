package workers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/JorgeIba/Hurion/app/batchschema"
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

		var (
			timer   *time.Timer
			timerCh <-chan time.Time
			pending bool
		)

		for {
			select {
			case <-w.updateEventChannel:
				pending = true
				if timer == nil {
					timer = time.NewTimer(w.DebounceDuration)
					timerCh = timer.C
				} else {
					// In Go 1.23+, timer.Reset directly drains and resets without requiring explicit channel drain boilerplate
					timer.Reset(w.DebounceDuration)
				}

			case <-timerCh:
				timer = nil
				timerCh = nil
				pending = false
				w.runExportWithTimeout()

			case <-w.shutdownSignal:
				if timer != nil {
					timer.Stop()
				}
				if pending {
					w.runExportWithTimeout()
				}
				return
			}
		}
	}()
}

func (w *ScoreboardExporterWorker) runExportWithTimeout() {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultExportTimeout)
	defer cancel()
	if err := w.ExportAll(ctx); err != nil {
		if logger.Log != nil {
			logger.Log.Errorf("error executing background scoreboard export: %v", err)
		}
	}
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
	if w.ScoreService == nil {
		return fmt.Errorf("score service is uninitialized")
	}
	if w.Storage == nil {
		return fmt.Errorf("storage provider is uninitialized")
	}

	classes, err := w.ClassService.GetAllClasses()
	if err != nil {
		return fmt.Errorf("failed to get classes: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("export context canceled: %w", err)
	}

	w.exportBatchSchemas(ctx)

	classesBytes, err := jsonexport.FormatClasses(classes)
	if err != nil {
		if logger.Log != nil {
			logger.Log.Errorf("failed to format classes JSON: %v", err)
		}
	} else if err := w.Storage.Save(ctx, "classes.json", classesBytes); err != nil {
		if logger.Log != nil {
			logger.Log.Errorf("failed to save classes.json: %v", err)
		}
	}

	for _, c := range classes {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("export context canceled: %w", err)
		}

		score, err := w.ScoreService.GetScoreByClassCode(c.ClassCode)
		if err != nil {
			if logger.Log != nil {
				logger.Log.Errorf("failed to get score for class %s: %v", c.ClassCode, err)
			}
			continue
		}

		jsonBytes, err := jsonexport.FormatScoreboard(c.ClassCode, c.Name, score)
		if err != nil {
			if logger.Log != nil {
				logger.Log.Errorf("failed to format scoreboard JSON for class %s: %v", c.ClassCode, err)
			}
			continue
		}

		filename := fmt.Sprintf("%s_score.json", c.ClassCode)
		if err := w.Storage.Save(ctx, filename, jsonBytes); err != nil {
			if logger.Log != nil {
				logger.Log.Errorf("failed to save scoreboard for class %s: %v", c.ClassCode, err)
			}
		}
	}

	return nil
}

// exportBatchSchemas publishes the exact schema files embedded in this bot build.
// Publication is best effort: a storage outage must never disable Discord commands.
func (w *ScoreboardExporterWorker) exportBatchSchemas(ctx context.Context) {
	files, err := batchschema.FS.ReadDir(".")
	if err != nil {
		if logger.Log != nil {
			logger.Log.Errorf("failed to list batch schemas: %v", err)
		}
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if err := ctx.Err(); err != nil {
			if logger.Log != nil {
				logger.Log.Errorf("stopped exporting batch schemas: %v", err)
			}
			return
		}

		contents, err := batchschema.FS.ReadFile(file.Name())
		if err != nil {
			if logger.Log != nil {
				logger.Log.Errorf("failed to read batch schema %s: %v", file.Name(), err)
			}
			continue
		}
		if err := w.Storage.Save(ctx, file.Name(), contents); err != nil && logger.Log != nil {
			logger.Log.Errorf("failed to save batch schema %s: %v", file.Name(), err)
		}
	}
}
