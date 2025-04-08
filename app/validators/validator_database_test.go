package validators

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/stretchr/testify/assert"
)

type MockDatastore struct {
	MockGetClassIDByClassCode   func(classCode string) (int64, error)
	MockIsUserRegisteredInClass func(userID string, classID int64) (bool, error)
	MockGetProblemByID          func(problemID int64) (*data.Problem, error)
}

func (m *MockDatastore) GetClassIDByClassCode(classCode string) (int64, error) {
	return m.MockGetClassIDByClassCode(classCode)
}

func (m *MockDatastore) IsUserRegisteredInClass(userID string, classID int64) (bool, error) {
	return m.MockIsUserRegisteredInClass(userID, classID)
}

func (m *MockDatastore) GetProblemByID(problemID int64) (*data.Problem, error) {
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

func TestExistsProblem_Yes(t *testing.T) {
	mockedDataStore := &MockDatastore{
		MockGetProblemByID: func(problemID int64) (*data.Problem, error) {
			return &data.Problem{}, nil
		},
	}

	validator := NewValidatorDB(mockedDataStore)

	isProblemOK, err := validator.ExistsProblem(123)
	assert.NoError(t, err)
	assert.True(t, isProblemOK)
}

func TestExistsProblem_No(t *testing.T) {
	mockedDataStore := &MockDatastore{
		MockGetProblemByID: func(problemID int64) (*data.Problem, error) {
			return &data.Problem{}, fmt.Errorf("Problem not found")
		},
	}

	validator := NewValidatorDB(mockedDataStore)

	isProblemOK, err := validator.ExistsProblem(123)
	assert.Error(t, err)
	assert.False(t, isProblemOK)
}

func TestExistsProblemInClass_Yes(t *testing.T) {
	mockedDataStore := &MockDatastore{
		MockGetProblemByID: func(problemID int64) (*data.Problem, error) {
			return &data.Problem{ClassID: 123}, nil
		},
	}

	validator := NewValidatorDB(mockedDataStore)

	isProblemOK, err := validator.ExistsProblemInClass(1, 123)
	assert.NoError(t, err)
	assert.True(t, isProblemOK)
}

func TestExistsProblemInClass_No(t *testing.T) {
	mockedDataStore := &MockDatastore{
		MockGetProblemByID: func(problemID int64) (*data.Problem, error) {
			return &data.Problem{ClassID: 777}, nil
		},
	}

	validator := NewValidatorDB(mockedDataStore)

	isProblemOK, err := validator.ExistsProblemInClass(1, 111)
	assert.NoError(t, err)
	assert.False(t, isProblemOK)
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
