package mentor

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCanManageACs(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name                   string
		mentor, admin, allowed bool
		mentorErr, adminErr    error
	}{
		{name: "mentor does not need an admin check", mentor: true, allowed: true},
		{name: "admin without mentor role is allowed", admin: true, allowed: true},
		{name: "neither role is denied"},
		{name: "mentor lookup failure stops authorization", mentorErr: errors.New("mentor lookup failed")},
		{name: "admin lookup failure stops authorization", adminErr: errors.New("admin lookup failed")},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			validator := mock_providers.NewMockDiscordValidator(gomock.NewController(t))
			validator.EXPECT().IsMentor("guild", "actor").Return(test.mentor, test.mentorErr)
			if !test.mentor && test.mentorErr == nil {
				validator.EXPECT().IsAdmin("guild", "actor").Return(test.admin, test.adminErr)
			}
			allowed, err := canManageACs(validator, "guild", "actor")
			assert.Equal(t, test.allowed, allowed)
			if test.mentorErr != nil {
				assert.ErrorIs(t, err, test.mentorErr)
			} else if test.adminErr != nil {
				assert.ErrorIs(t, err, test.adminErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
