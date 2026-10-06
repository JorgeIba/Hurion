package mentor

import (
	"time"

	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/maypok86/otter/v2"
)

const (
	createProblemDraftCapacity = 100
	createProblemDraftTTL      = 10 * time.Minute
)

type createProblemDraftCacheKey struct {
	guildID   string
	userID    string
	classCode string
}

type createProblemDraftCacheValue = discord.CreateProblemModal

type createProblemDraftCache struct {
	cache *otter.Cache[createProblemDraftCacheKey, createProblemDraftCacheValue]
}

func newCreateProblemDraftCache(clock otter.Clock) *createProblemDraftCache {
	return &createProblemDraftCache{cache: otter.Must(&otter.Options[createProblemDraftCacheKey, createProblemDraftCacheValue]{
		MaximumSize:      createProblemDraftCapacity,
		ExpiryCalculator: createProblemDraftExpiry{},
		Clock:            clock,
	})}
}

func (c *createProblemDraftCache) saveFailedDraft(key createProblemDraftCacheKey, modal createProblemDraftCacheValue) {
	c.cache.Set(key, modal)
}

func (c *createProblemDraftCache) loadFailedDraft(key createProblemDraftCacheKey) (createProblemDraftCacheValue, bool) {
	return c.cache.GetIfPresent(key)
}

func (c *createProblemDraftCache) consumeDraft(key createProblemDraftCacheKey) {
	c.cache.Invalidate(key)
}

type createProblemDraftExpiry struct{}

func (createProblemDraftExpiry) ExpireAfterCreate(otter.Entry[createProblemDraftCacheKey, createProblemDraftCacheValue]) time.Duration {
	return createProblemDraftTTL
}

func (createProblemDraftExpiry) ExpireAfterUpdate(otter.Entry[createProblemDraftCacheKey, createProblemDraftCacheValue], createProblemDraftCacheValue) time.Duration {
	return createProblemDraftTTL
}

func (createProblemDraftExpiry) ExpireAfterRead(entry otter.Entry[createProblemDraftCacheKey, createProblemDraftCacheValue]) time.Duration {
	return entry.ExpiresAfter()
}
