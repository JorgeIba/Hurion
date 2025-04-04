package validators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockDatastore struct {
	MockGetClassIDByClassCode   func(classCode string) (int64, error)
	MockIsUserRegisteredInClass func(userID string, classID int64) (bool, error)
}

func (m *MockDatastore) GetClassIDByClassCode(classCode string) (int64, error) {
	return m.MockGetClassIDByClassCode(classCode)
}

func (m *MockDatastore) IsUserRegisteredInClass(userID string, classID int64) (bool, error) {
	return m.MockIsUserRegisteredInClass(userID, classID)
}

func TestIsEnrolled_Yes(t *testing.T) {
	mockedDataStore := &MockDatastore{
		MockGetClassIDByClassCode: func(classCode string) (int64, error) {
			return int64(1), nil
		},
		MockIsUserRegisteredInClass: func(userID string, classID int64) (bool, error) {
			return true, nil
		},
	}

	validator := &ValidatorDB{db: mockedDataStore}

	isMyBoyEnrolled, err := validator.IsEnrolled("Jorge", "1")
	assert.NoError(t, err)
	assert.True(t, isMyBoyEnrolled)
}

func TestIsEnrolled_No(t *testing.T) {
	mockedDataStore := &MockDatastore{
		MockGetClassIDByClassCode: func(classCode string) (int64, error) {
			return int64(1), nil
		},
		MockIsUserRegisteredInClass: func(userID string, classID int64) (bool, error) {
			return false, nil
		},
	}

	validator := &ValidatorDB{db: mockedDataStore}

	isMyBoyEnrolled, err := validator.IsEnrolled("Jorge", "1")
	assert.NoError(t, err)
	assert.False(t, isMyBoyEnrolled)
}
