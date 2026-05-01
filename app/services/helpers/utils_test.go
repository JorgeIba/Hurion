package helpers

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mock_data"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestExistsProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		problemID int64
		setupMock func(*mock_data.MockDataStore)
		want      bool
		wantErr   bool
	}{
		{
			name:      "exists",
			problemID: 1,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{ID: 1}, nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "not_found",
			problemID: 1,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{}, &dataerr.NotFoundError{})
			},
			want:    false,
			wantErr: false,
		},
		{
			name:      "db_error",
			problemID: 1,
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(1)).Return(data.Problem{}, errors.New("db error"))
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			got, err := ExistsProblem(mockStore, tt.problemID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIsClassCodeValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		classCode string
		setupMock func(*mock_data.MockDataStore)
		want      bool
		wantErr   bool
	}{
		{
			name:      "valid",
			classCode: "c1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("c1").Return(data.Class{ClassCode: "c1"}, nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "invalid",
			classCode: "c1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("c1").Return(data.Class{}, &dataerr.NotFoundError{})
			},
			want:    false,
			wantErr: false,
		},
		{
			name:      "db_error",
			classCode: "c1",
			setupMock: func(m *mock_data.MockDataStore) {
				m.EXPECT().GetClassByClassCode("c1").Return(data.Class{}, errors.New("db error"))
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_data.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			got, err := IsClassCodeValid(mockStore, tt.classCode)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
