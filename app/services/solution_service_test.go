package services

import (
	"errors"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSolutionService_AddACToUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userID      string
		problemID   int64
		setupMock   func(*mocks.MockDataStore)
		wantErr     bool
		wantErrType error
	}{
		{
			name:      "success - AC added",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				// helpers.ExistsProblem
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100, ClassID: 10}, nil).Times(2)
				// helpers.CanBroACTheProblem -> IsUserIDEnrolledByClassID
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(true, nil)
				// helpers.CanBroACTheProblem -> HasUserSolvedProblem
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(false, nil)
				// SolutionService.AddACToUser
				m.EXPECT().AddAC("u1", int64(100)).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "problem does not exist",
			userID:    "u1",
			problemID: 404,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(404)).Return(data.Problem{}, &dataerr.NotFoundError{})
			},
			wantErr:     true,
			wantErrType: &serviceerr.ProblemNotFoundError{},
		},
		{
			name:      "user not enrolled",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100, ClassID: 10}, nil).Times(2)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(false, nil)
			},
			wantErr:     true,
			wantErrType: &serviceerr.UserNotEnrolledError{},
		},
		{
			name:      "already solved",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100, ClassID: 10}, nil).Times(2)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(true, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(true, nil)
			},
			wantErr:     true,
			wantErrType: &serviceerr.ConflictError{},
		},
		{
			name:      "AddAC generic error",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100, ClassID: 10}, nil).Times(2)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(true, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(false, nil)
				m.EXPECT().AddAC("u1", int64(100)).Return(errors.New("db fail"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := &SolutionService{Datastore: mockStore}
			err := svc.AddACToUser(tt.userID, tt.problemID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.IsType(t, tt.wantErrType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSolutionService_UndoACByProblemAndUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userID    string
		problemID int64
		setupMock func(*mocks.MockDataStore)
		wantErr   bool
	}{
		{
			name:      "success - AC removed",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100}, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(true, nil)
				m.EXPECT().RemoveAC("u1", int64(100)).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "not solved",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100}, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(false, nil)
			},
			wantErr: true,
		},
		{
			name:      "RemoveAC fail",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100}, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(true, nil)
				m.EXPECT().RemoveAC("u1", int64(100)).Return(errors.New("db fail"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := &SolutionService{Datastore: mockStore}
			err := svc.UndoACByProblemAndUserID(tt.problemID, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSolutionService_AddACToUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userName  string
		problemID int64
		setupMock func(*mocks.MockDataStore)
		wantErr   bool
	}{
		{
			name:      "success",
			userName:  "user1",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetUserByUserName("user1").Return(data.User{UserID: "u1"}, nil)
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100, ClassID: 10}, nil).Times(2)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(true, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(false, nil)
				m.EXPECT().AddAC("u1", int64(100)).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "user not found",
			userName:  "ghost",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetUserByUserName("ghost").Return(data.User{}, &dataerr.NotFoundError{})
			},
			wantErr: true,
		},
		{
			name:      "generic error GetUserByUserName",
			userName:  "user1",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetUserByUserName("user1").Return(data.User{}, errors.New("db fail"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := &SolutionService{Datastore: mockStore}
			err := svc.AddACToUsername(tt.userName, tt.problemID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSolutionService_UndoACByProblemAndUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userName  string
		problemID int64
		setupMock func(*mocks.MockDataStore)
		wantErr   bool
	}{
		{
			name:      "success",
			userName:  "user1",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetUserByUserName("user1").Return(data.User{UserID: "u1"}, nil)
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100}, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(true, nil)
				m.EXPECT().RemoveAC("u1", int64(100)).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "user not found",
			userName:  "ghost",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetUserByUserName("ghost").Return(data.User{}, &dataerr.NotFoundError{})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := &SolutionService{Datastore: mockStore}
			err := svc.UndoACByProblemAndUsername(tt.problemID, tt.userName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSolutionService_AddPendingACOfUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		messageID string
		userID    string
		problemID int64
		setupMock func(*mocks.MockDataStore)
		wantErr   bool
	}{
		{
			name:      "success - pending AC added",
			messageID: "m1",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100, ClassID: 10}, nil).Times(2)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(true, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(false, nil)
				m.EXPECT().AddPendingConfirmation(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "problem not present",
			messageID: "m1",
			userID:    "u1",
			problemID: 404,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(404)).Return(data.Problem{}, &dataerr.NotFoundError{})
			},
			wantErr: true,
		},
		{
			name:      "cannot AC",
			messageID: "m1",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100, ClassID: 10}, nil).Times(2)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(false, nil)
			},
			wantErr: true,
		},
		{
			name:      "AddPendingConfirmation fail",
			messageID: "m1",
			userID:    "u1",
			problemID: 100,
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100, ClassID: 10}, nil).Times(2)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(true, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(false, nil)
				m.EXPECT().AddPendingConfirmation(gomock.Any()).Return(errors.New("db fail"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := &SolutionService{Datastore: mockStore}
			err := svc.AddPendingACOfUserID(tt.messageID, tt.userID, tt.problemID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSolutionService_ExistsPendingConfirmation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		messageID string
		setupMock func(*mocks.MockDataStore)
		want      bool
		wantErr   bool
	}{
		{
			name:      "exists",
			messageID: "m1",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().ExistsPendingConfirmation("m1").Return(true, nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "error",
			messageID: "m1",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().ExistsPendingConfirmation("m1").Return(false, errors.New("db fail"))
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
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := &SolutionService{Datastore: mockStore}
			got, err := svc.ExistsPendingConfirmation(tt.messageID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSolutionService_GetPendingConfirmationByMessageID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		messageID string
		setupMock func(*mocks.MockDataStore)
		wantErr   bool
	}{
		{
			name:      "success",
			messageID: "m1",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetPendingConfirmation("m1").Return("u1", int64(100), nil)
			},
			wantErr: false,
		},
		{
			name:      "not found",
			messageID: "m1",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetPendingConfirmation("m1").Return("", int64(0), &dataerr.NotFoundError{})
			},
			wantErr: true,
		},
		{
			name:      "generic error",
			messageID: "m1",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetPendingConfirmation("m1").Return("", int64(0), errors.New("db fail"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := &SolutionService{Datastore: mockStore}
			res, err := svc.GetPendingConfirmationByMessageID(tt.messageID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.messageID, res.MessageID)
			}
		})
	}
}

func TestSolutionService_AcceptACProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		messageID string
		setupMock func(*mocks.MockDataStore)
		wantErr   bool
	}{
		{
			name:      "success",
			messageID: "m1",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetPendingConfirmation("m1").Return("u1", int64(100), nil)
				m.EXPECT().RemovePendingConfirmation("m1").Return(nil)
				// From AddACToUser
				m.EXPECT().GetProblemByID(int64(100)).Return(data.Problem{ID: 100, ClassID: 10}, nil).Times(2)
				m.EXPECT().IsUserRegisteredInClass("u1", int64(10)).Return(true, nil)
				m.EXPECT().HasUserSolvedProblem("u1", int64(100)).Return(false, nil)
				m.EXPECT().AddAC("u1", int64(100)).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "error GetPendingConfirmationByMessageID",
			messageID: "m1",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetPendingConfirmation("m1").Return("", int64(0), errors.New("db fail"))
			},
			wantErr: true,
		},
		{
			name:      "error RemovePendingConfirmation",
			messageID: "m1",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().GetPendingConfirmation("m1").Return("u1", int64(100), nil)
				m.EXPECT().RemovePendingConfirmation("m1").Return(errors.New("db fail"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := &SolutionService{Datastore: mockStore}
			err := svc.AcceptACProposal(tt.messageID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSolutionService_RejectACProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		messageID string
		setupMock func(*mocks.MockDataStore)
		wantErr   bool
	}{
		{
			name:      "success",
			messageID: "m1",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().RemovePendingConfirmation("m1").Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "error",
			messageID: "m1",
			setupMock: func(m *mocks.MockDataStore) {
				m.EXPECT().RemovePendingConfirmation("m1").Return(errors.New("db fail"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mocks.NewMockDataStore(ctrl)
			tt.setupMock(mockStore)

			svc := &SolutionService{Datastore: mockStore}
			err := svc.RejectACProposal(tt.messageID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
