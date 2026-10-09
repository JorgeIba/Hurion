package service_model

import (
	"errors"
	"strconv"
	"strings"
)

const MaxBatchOperations = 100

// BatchEntryError uses one-based entry numbers; zero identifies a batch-wide error.
type BatchEntryError struct {
	Entry int
	Err   error
}

type CreateACParams struct {
	UserID    string
	ProblemID int64
}

// Prepared values are read-only snapshots for preview. Targets, permissions and
// conflicts must be checked again before executing the batch.
type PreparedProblem struct {
	Entry  int
	Params CreateProblemParams
	Class  Class
}

type PreparedAC struct {
	Entry   int
	User    User
	Problem Problem
	Class   Class
	// AlreadyRecorded marks an entry omitted from writes and reported in the preview.
	AlreadyRecorded bool
}

// NormalizeACUserID adapts textual IDs without validating whether an AC is valid.
func NormalizeACUserID(userID string) string {
	userID = strings.TrimSpace(userID)
	if id, err := strconv.ParseUint(userID, 10, 64); err == nil {
		return strconv.FormatUint(id, 10)
	}
	return userID
}

func NormalizeAndValidateCreateACParams(input CreateACParams) (CreateACParams, error) {
	input.UserID = NormalizeACUserID(input.UserID)
	id, err := strconv.ParseUint(input.UserID, 10, 64)

	var issues []error
	if err != nil || id == 0 {
		issues = append(issues, errors.New("user_id must be a positive numeric Discord ID"))
	}

	if input.ProblemID <= 0 {
		issues = append(issues, errors.New("problem_id must be positive"))
	}

	return input, errors.Join(issues...)
}
