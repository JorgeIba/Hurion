package validators

import (
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/stretchr/testify/assert"
)

type MockDatastore struct {
	MockGetClassIDByClassCode   func(classCode string) (int64, error)
	MockIsUserRegisteredInClass func(userID string, classID int64) (bool, error)
	MockGetProblemByID          func(problemID int64) (data.Problem, error)
}

func (m *MockDatastore) GetClassIDByClassCode(classCode string) (int64, error) {
	return m.MockGetClassIDByClassCode(classCode)
}

func (m *MockDatastore) IsUserRegisteredInClass(userID string, classID int64) (bool, error) {
	return m.MockIsUserRegisteredInClass(userID, classID)
}

func (m *MockDatastore) GetProblemByID(problemID int64) (data.Problem, error) {
	return m.MockGetProblemByID(problemID)
}

func TestIsEnrolledByClassCode_Yes(t *testing.T) {
	mockedDataStore := &MockDatastore{
		MockGetClassIDByClassCode: func(classCode string) (int64, error) {
			return int64(1), nil
		},
		MockIsUserRegisteredInClass: func(userID string, classID int64) (bool, error) {
			return true, nil
		},
	}

	validator := NewValidatorDB(mockedDataStore)

	isMyBoyEnrolled, err := validator.IsEnrolledByClassCode("Jorge", "1")
	assert.NoError(t, err)
	assert.True(t, isMyBoyEnrolled)
}

func TestIsEnrolledByClassCode_No(t *testing.T) {
	mockedDataStore := &MockDatastore{
		MockGetClassIDByClassCode: func(classCode string) (int64, error) {
			return int64(1), nil
		},
		MockIsUserRegisteredInClass: func(userID string, classID int64) (bool, error) {
			return false, nil
		},
	}

	validator := NewValidatorDB(mockedDataStore)

	isMyBoyEnrolled, err := validator.IsEnrolledByClassCode("Jorge", "1")
	assert.NoError(t, err)
	assert.False(t, isMyBoyEnrolled)
}

func TestIsEnrolledByClassID_Yes(t *testing.T) {
	mockedDataStore := &MockDatastore{
		MockIsUserRegisteredInClass: func(userID string, classID int64) (bool, error) {
			return true, nil
		},
	}

	validator := NewValidatorDB(mockedDataStore)

	isMyBoyEnrolled, err := validator.IsEnrolledByClassID("Jorge", 1)
	assert.NoError(t, err)
	assert.True(t, isMyBoyEnrolled)
}

func TestIsEnrolledByClassID_No(t *testing.T) {
	mockedDataStore := &MockDatastore{
		MockIsUserRegisteredInClass: func(userID string, classID int64) (bool, error) {
			return false, nil
		},
	}

	validator := NewValidatorDB(mockedDataStore)

	isMyBoyEnrolled, err := validator.IsEnrolledByClassID("Jorge", 1)
	assert.NoError(t, err)
	assert.False(t, isMyBoyEnrolled)
}
