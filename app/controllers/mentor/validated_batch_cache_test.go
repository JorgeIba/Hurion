package mentor

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
)

func TestValidatedBatchExpiresWithoutRenewingOnReads(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		after       time.Duration
		wantPresent bool
	}{
		{name: "before five minutes", after: 5*time.Minute - time.Nanosecond, wantPresent: true},
		{name: "at five minutes", after: 5 * time.Minute},
		{name: "after five minutes", after: 6 * time.Minute},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			clock := &draftClock{}
			clock.now.Store(time.Now().UnixNano())
			cache := newValidatedProblemBatchCache(clock)
			key := validatedBatchCacheKey{guildID: "guild", userID: "mentor", batchID: "batch"}
			saved := cache.saveValidatedBatch(key, []service_model.PreparedProblem{{Entry: 1}})
			assert.True(t, saved)
			clock.now.Add(int64(4 * time.Minute))
			_, present := cache.loadValidatedBatch(key)
			assert.True(t, present, "a read must not extend the original deadline")
			clock.now.Add(int64(test.after - 4*time.Minute))
			_, found := cache.consumeBatch(key)
			assert.Equal(t, test.wantPresent, found)
			_, present = cache.loadValidatedBatch(key)
			assert.False(t, present)
		})
	}
}

func TestValidatedBatchIsBoundToAuthorGuildAndID(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		change func(*validatedBatchCacheKey)
	}{
		{name: "other author", change: func(key *validatedBatchCacheKey) { key.userID = "other" }},
		{name: "other guild", change: func(key *validatedBatchCacheKey) { key.guildID = "other" }},
		{name: "other batch", change: func(key *validatedBatchCacheKey) { key.batchID = "other" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cache := newValidatedProblemBatchCache(nil)
			value := []service_model.PreparedProblem{{Entry: 1}}
			key := validatedBatchCacheKey{guildID: "guild", userID: "mentor", batchID: "batch"}
			saved := cache.saveValidatedBatch(key, value)
			assert.True(t, saved)
			other := key
			test.change(&other)
			_, found := cache.consumeBatch(other)
			assert.False(t, found)
			got, found := cache.consumeBatch(key)
			assert.True(t, found)
			assert.Equal(t, value, got)
		})
	}
}

func TestValidatedBatchHasOnlyOneConcurrentClaimant(t *testing.T) {
	t.Parallel()
	cache := newValidatedProblemBatchCache(nil)
	key := validatedBatchCacheKey{guildID: "guild", userID: "mentor", batchID: "batch"}
	saved := cache.saveValidatedBatch(key, []service_model.PreparedProblem{{Entry: 1}})
	assert.True(t, saved)
	var winners atomic.Int32
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			if _, found := cache.consumeBatch(key); found {
				winners.Add(1)
			}
		})
	}
	workers.Wait()
	assert.Equal(t, int32(1), winners.Load())
}

func TestValidatedBatchCachesAreIndependentAndBounded(t *testing.T) {
	t.Parallel()
	problems := newValidatedProblemBatchCache(nil)
	acs := newValidatedACBatchCache(nil)
	edits := newValidatedProblemEditBatchCache(nil)
	key := validatedBatchCacheKey{guildID: "guild", userID: "mentor", batchID: "same-id"}
	assert.True(t, problems.saveValidatedBatch(key, []service_model.PreparedProblem{{Entry: 1}}))
	assert.True(t, acs.saveValidatedBatch(key, []service_model.PreparedAC{{Entry: 2}}))
	assert.True(t, edits.saveValidatedBatch(key, []service_model.PreparedProblemPatch{{Entry: 3}}))
	_, found := problems.consumeBatch(key)
	assert.True(t, found)
	value, found := acs.loadValidatedBatch(key)
	assert.True(t, found)
	assert.Equal(t, 2, value[0].Entry)
	for index := range 200 {
		problems.saveValidatedBatch(validatedBatchCacheKey{batchID: fmt.Sprint(index)}, []service_model.PreparedProblem{{Entry: index}})
	}
	problems.cache.CleanUp()
	assert.LessOrEqual(t, problems.cache.EstimatedSize(), 50)
	assert.Equal(t, uint64(50), acs.cache.GetMaximum())
	value, found = acs.consumeBatch(key)
	assert.True(t, found, "filling the problem cache must not evict an AC batch")
	editValue, found := edits.consumeBatch(key)
	assert.True(t, found, "consuming a create or AC batch must not consume an edit batch")
	assert.Equal(t, 3, editValue[0].Entry)
	assert.Equal(t, 2, value[0].Entry)
}

func TestValidatedBatchesForSameActorRemainIndependent(t *testing.T) {
	t.Parallel()
	cache := newValidatedProblemBatchCache(nil)
	first := validatedBatchCacheKey{guildID: "guild", userID: "mentor", batchID: "first"}
	second := first
	second.batchID = "second"
	assert.True(t, cache.saveValidatedBatch(first, []service_model.PreparedProblem{{Entry: 1}}))
	assert.True(t, cache.saveValidatedBatch(second, []service_model.PreparedProblem{{Entry: 2}}))
	firstBatch, found := cache.consumeBatch(first)
	assert.True(t, found)
	assert.Equal(t, 1, firstBatch[0].Entry)
	secondBatch, found := cache.consumeBatch(second)
	assert.True(t, found)
	assert.Equal(t, 2, secondBatch[0].Entry)
}
