package mentor

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type classPermissionValidator struct {
	providers.DiscordValidator
	check func(string, string, string) (bool, error)
}

func (v classPermissionValidator) CanManageClass(guildID, userID, classCode string) (bool, error) {
	return v.check(guildID, userID, classCode)
}

func TestAuthorizeMentorTargets(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name                               string
		target                             mentorTarget
		denied                             string
		lookupErr, classErr, permissionErr bool
		wantErr                            bool
	}{
		{name: "current class allowed", target: mentorTarget{problemID: 42}},
		{name: "current class denied", target: mentorTarget{problemID: 42}, denied: "source", wantErr: true},
		{name: "move destination denied", target: mentorTarget{problemID: 42, classCode: " destination "}, denied: "destination", wantErr: true},
		{name: "move source denied", target: mentorTarget{problemID: 42, classCode: "destination"}, denied: "source", wantErr: true},
		{name: "both classes allowed", target: mentorTarget{problemID: 42, classCode: "destination"}},
		{name: "problem lookup fails", target: mentorTarget{problemID: 42}, lookupErr: true, wantErr: true},
		{name: "class lookup fails", target: mentorTarget{problemID: 42}, classErr: true, wantErr: true},
		{name: "permission lookup fails", target: mentorTarget{classCode: "destination"}, permissionErr: true, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_data.NewMockDataStore(ctrl)
			failure := errors.New("lookup unavailable")
			if tt.target.problemID != 0 {
				var err error
				if tt.lookupErr {
					err = failure
				}
				store.EXPECT().GetProblemByID(int64(42)).Return(data.Problem{ID: 42, ClassID: 7}, err)
				if !tt.lookupErr {
					var classErr error
					if tt.classErr {
						classErr = failure
					}
					store.EXPECT().GetClassByClassID(int64(7)).Return(data.Class{ID: 7, ClassCode: "source"}, classErr)
				}
			}
			validator := classPermissionValidator{check: func(guild, actor, class string) (bool, error) {
				assert.Equal(t, "guild", guild)
				assert.Equal(t, "actor", actor)
				if tt.permissionErr {
					return false, failure
				}
				return class != tt.denied, nil
			}}
			err := authorizeMentorTargets(validator, services.NewServices(store), "guild", "actor", tt.target)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
