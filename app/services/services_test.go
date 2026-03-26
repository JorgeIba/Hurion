package services

import (
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mock_data"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewServices(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_data.NewMockDataStore(ctrl)
	
	services := NewServices(mockStore)
	
	assert.NotNil(t, services)
	assert.NotNil(t, services.User)
	assert.NotNil(t, services.Enrollment)
	assert.NotNil(t, services.Class)
	assert.NotNil(t, services.Problem)
	assert.NotNil(t, services.Solution)
	assert.NotNil(t, services.Score)
}
