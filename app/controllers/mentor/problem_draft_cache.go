package mentor

import (
	"time"

	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/maypok86/otter/v2"
)

const (
	problemDraftCapacity = 100
	problemDraftTTL      = 10 * time.Minute
)

type createProblemDraftCacheKey struct {
	guildID   string
	userID    string
	classCode string
}

type editProblemDraftCacheKey struct {
	guildID   string
	userID    string
	problemID int64
	classCode string
}

type createProblemDraftCache = problemDraftCache[createProblemDraftCacheKey, discord.CreateProblemModal]
type editProblemDraftCache = problemDraftCache[editProblemDraftCacheKey, discord.EditProblemModal]

type problemDraftCache[K comparable, V any] struct {
	cache *otter.Cache[K, V]
}

func newProblemDraftCache[K comparable, V any](clock otter.Clock) *problemDraftCache[K, V] {
	return &problemDraftCache[K, V]{cache: otter.Must(&otter.Options[K, V]{
		MaximumSize:      problemDraftCapacity,
		ExpiryCalculator: otter.ExpiryWriting[K, V](problemDraftTTL),
		Clock:            clock,
	})}
}

func newCreateProblemDraftCache(clock otter.Clock) *createProblemDraftCache {
	return newProblemDraftCache[createProblemDraftCacheKey, discord.CreateProblemModal](clock)
}

func newEditProblemDraftCache(clock otter.Clock) *editProblemDraftCache {
	return newProblemDraftCache[editProblemDraftCacheKey, discord.EditProblemModal](clock)
}

// saveFailedDraft reports availability immediately after writing, not future retention.
func (c *problemDraftCache[K, V]) saveFailedDraft(key K, modal V) bool {
	c.cache.Set(key, modal)
	_, available := c.cache.GetIfPresent(key)
	return available
}

func (c *problemDraftCache[K, V]) loadFailedDraft(key K) (V, bool) {
	return c.cache.GetIfPresent(key)
}

func (c *problemDraftCache[K, V]) consumeDraft(key K) {
	c.cache.Invalidate(key)
}
