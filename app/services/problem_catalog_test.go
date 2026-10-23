package services

import (
	"errors"
	"testing"

	authtest "github.com/JorgeIba/Hurion/app/authorization/authtest"
	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestProblemService_GetProblemsByClass(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		setupMock  func(*mock_data.MockDataStore)
		assertFunc func(*testing.T, []service_model.Problem, error)
	}{
		{
			name: "orders repository results numerically by ID",
			setupMock: func(store *mock_data.MockDataStore) {
				store.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 7}, nil)
				store.EXPECT().GetProblemsByClass(int64(7)).Return([]data.Problem{
					{ID: 10, ClassID: 7, Name: "Ten"},
					{ID: 2, ClassID: 7, Name: "Two"},
					{ID: 1, ClassID: 7, Name: "One"},
				}, nil)
			},
			assertFunc: func(t *testing.T, problems []service_model.Problem, err error) {
				assert.NoError(t, err)
				assert.Equal(t, []service_model.Problem{
					{ID: 1, ClassID: 7, Name: "One"},
					{ID: 2, ClassID: 7, Name: "Two"},
					{ID: 10, ClassID: 7, Name: "Ten"},
				}, problems)
			},
		},
		{
			name: "class does not exist",
			setupMock: func(store *mock_data.MockDataStore) {
				store.EXPECT().GetClassByClassCode("basicos").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			assertFunc: func(t *testing.T, problems []service_model.Problem, err error) {
				assert.Empty(t, problems)
				assert.IsType(t, &serviceerr.ClassCodeNotFoundError{}, err)
			},
		},
		{
			name: "class read fails",
			setupMock: func(store *mock_data.MockDataStore) {
				store.EXPECT().GetClassByClassCode("basicos").Return(data.Class{}, errors.New("read failed"))
			},
			assertFunc: func(t *testing.T, problems []service_model.Problem, err error) {
				assert.Empty(t, problems)
				assert.IsType(t, &serviceerr.InternalError{}, err)
			},
		},
		{
			name: "catalog read fails",
			setupMock: func(store *mock_data.MockDataStore) {
				store.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 7}, nil)
				store.EXPECT().GetProblemsByClass(int64(7)).Return(nil, errors.New("read failed"))
			},
			assertFunc: func(t *testing.T, problems []service_model.Problem, err error) {
				assert.Empty(t, problems)
				assert.IsType(t, &serviceerr.InternalError{}, err)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mock_data.NewMockDataStore(gomock.NewController(t))
			test.setupMock(store)
			problems, err := NewProblemService(store, authtest.Allow{}).GetProblemsByClass("basicos")
			test.assertFunc(t, problems, err)
		})
	}
}
