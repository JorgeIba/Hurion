package mentor

import (
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type draftClock struct {
	now atomic.Int64
}

func (c *draftClock) NowNano() int64                      { return c.now.Load() }
func (c *draftClock) Tick(time.Duration) <-chan time.Time { return nil }

func TestCreateProblemDraftExpiresTenMinutesAfterLastFailure(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		failAgain   bool
		after       time.Duration
		wantPresent bool
	}{
		{name: "available just before expiration", after: 10*time.Minute - time.Nanosecond, wantPresent: true},
		{name: "unavailable at expiration", after: 10 * time.Minute},
		{name: "another failure replaces values and renews expiration", failAgain: true, after: 9 * time.Minute, wantPresent: true},
		{name: "replacement also expires after ten minutes", failAgain: true, after: 10 * time.Minute},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			clock := &draftClock{}
			clock.now.Store(time.Now().UnixNano())
			cache := newCreateProblemDraftCache(clock)
			key := createProblemDraftCacheKey{guildID: "guild", userID: "mentor", classCode: "basicos_1"}
			modal := interactiontransport.CreateProblemModal{ClassCode: key.classCode, ProblemModalFields: interactiontransport.ProblemModalFields{Name: "Original", URL: "invalid"}}
			assert.True(t, cache.saveFailedDraft(key, modal), "a new draft is available after writing")
			if test.failAgain {
				clock.now.Add(int64(9 * time.Minute))
				modal.Name = "Corrected name"
				assert.True(t, cache.saveFailedDraft(key, modal), "replacing a draft also reports availability")
			}
			clock.now.Add(int64(test.after))
			got, found := cache.loadFailedDraft(key)
			assert.Equal(t, test.wantPresent, found)
			if found {
				assert.Equal(t, modal, got)
			}
		})
	}
}

func TestCreateProblemDraftIsIsolatedByServerAuthorAndClass(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		key  createProblemDraftCacheKey
	}{
		{name: "other server", key: createProblemDraftCacheKey{guildID: "other", userID: "mentor", classCode: "basicos_1"}},
		{name: "other author", key: createProblemDraftCacheKey{guildID: "guild", userID: "other", classCode: "basicos_1"}},
		{name: "other class", key: createProblemDraftCacheKey{guildID: "guild", userID: "mentor", classCode: "basicos_2"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cache := newCreateProblemDraftCache(nil)
			key := createProblemDraftCacheKey{guildID: "guild", userID: "mentor", classCode: "basicos_1"}
			modal := interactiontransport.CreateProblemModal{ClassCode: key.classCode, ProblemModalFields: interactiontransport.ProblemModalFields{Name: "My problem"}}
			cache.saveFailedDraft(key, modal)
			_, found := cache.loadFailedDraft(test.key)
			assert.False(t, found)
			got, found := cache.loadFailedDraft(key)
			assert.True(t, found)
			assert.Equal(t, modal, got)
		})
	}
}
