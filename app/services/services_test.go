package services

import (
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewServices(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mocks.NewMockDataStore(ctrl)

	svcs := NewServices(mockStore)

	assert.NotNil(t, svcs.User)
	assert.NotNil(t, svcs.Enrollment)
	assert.NotNil(t, svcs.Class)
	assert.NotNil(t, svcs.Problem)
	assert.NotNil(t, svcs.Solution)
	assert.NotNil(t, svcs.Score)
	assert.Equal(t, mockStore, svcs.User.Datastore)
}
