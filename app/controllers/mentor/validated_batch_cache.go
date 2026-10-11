package mentor

import (
	"time"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/maypok86/otter/v2"
)

const (
	validatedBatchCapacity = 50
	validatedBatchTTL      = 5 * time.Minute
)

type validatedBatchCacheKey struct {
	guildID string
	userID  string
	batchID string
}

type validatedProblemBatchCache = validatedBatchCache[service_model.PreparedProblem]
type validatedACBatchCache = validatedBatchCache[service_model.PreparedAC]

// A validated snapshot is not authorization to execute it later.
type validatedBatchCache[T any] struct {
	cache *otter.Cache[validatedBatchCacheKey, []T]
}

func newValidatedBatchCache[T any](clock otter.Clock) *validatedBatchCache[T] {
	return &validatedBatchCache[T]{cache: otter.Must(&otter.Options[validatedBatchCacheKey, []T]{
		MaximumSize:      validatedBatchCapacity,
		ExpiryCalculator: otter.ExpiryWriting[validatedBatchCacheKey, []T](validatedBatchTTL),
		Clock:            clock,
	})}
}

func newValidatedProblemBatchCache(clock otter.Clock) *validatedProblemBatchCache {
	return newValidatedBatchCache[service_model.PreparedProblem](clock)
}

func newValidatedACBatchCache(clock otter.Clock) *validatedACBatchCache {
	return newValidatedBatchCache[service_model.PreparedAC](clock)
}

// saveValidatedBatch reports availability immediately after writing, not future retention.
func (c *validatedBatchCache[T]) saveValidatedBatch(key validatedBatchCacheKey, batch []T) bool {
	c.cache.Set(key, batch)
	_, available := c.cache.GetIfPresent(key)
	return available
}

func (c *validatedBatchCache[T]) loadValidatedBatch(key validatedBatchCacheKey) ([]T, bool) {
	return c.cache.GetIfPresent(key)
}

// consumeBatch claims an unexpired snapshot once, even with concurrent clicks.
func (c *validatedBatchCache[T]) consumeBatch(key validatedBatchCacheKey) ([]T, bool) {
	var claimed []T
	var available bool
	c.cache.Compute(key, func(batch []T, found bool) ([]T, otter.ComputeOp) {
		if !found {
			return nil, otter.CancelOp
		}
		claimed, available = batch, true
		return nil, otter.InvalidateOp
	})
	return claimed, available
}

type validatedProblemEditBatchCache = validatedBatchCache[service_model.PreparedProblemPatch]

func newValidatedProblemEditBatchCache(clock otter.Clock) *validatedProblemEditBatchCache {
	return newValidatedBatchCache[service_model.PreparedProblemPatch](clock)
}
