package service_model

import (
	jsonv2 "encoding/json/v2"
	"fmt"
	"strings"
	"time"
)

// ============================================================================
// Core Service Models (Domain Entities)
// ============================================================================

// User represents a user in the system
type User struct {
	UserID      string
	DisplayName string
}

// Principal represents the user ACTING on the system right now,
// containing all their verified security credentials.
type Principal struct {
	User
	Roles []string
}

// Problem represents a problem within a class
type Problem struct {
	ID         int64
	Name       string
	ClassID    int64
	URL        string
	Difficulty *string
	Topic      *string
	Priority   *string
	CreatedAt  time.Time
}

// Class represents a class or group of problems
type Class struct {
	ID        int64
	ClassCode string
	Name      string
}

// Classroom represents a location where a class is taught
type Classroom struct {
	RoomID    string
	ClassCode string
}

// ClassAccessRole represents a role that grants access to a class
type ClassAccessRole struct {
	RoleID    string
	ClassCode string
}

// SolvedProblem represents a record of a user solving a problem
type SolvedProblem struct {
	UserID    string
	ProblemID int64
	SolvedAt  time.Time
}

// PendingConfirmation represents a pending AC proposal awaiting mentor approval
type PendingConfirmation struct {
	MessageID string
	UserID    string
	ProblemID int64
}

// ============================================================================
// Request Parameters
// ============================================================================

type CreateACParams struct {
	UserID    string
	ProblemID int64
}

// ProblemFields contains the editable values shared by creation and updates.
type ProblemFields struct {
	Name       string
	URL        string
	Difficulty *string
	Topic      *string
	Priority   *string
}

// CreateProblemParams represents the parameters needed to create a new problem.
// CreatedAt is intentionally absent; the database assigns it on insertion.
type CreateProblemParams struct {
	ProblemFields
	ClassCode string
}

// ReplaceProblemParams identifies the target and supplies all replacement values,
// including class membership. Nil Difficulty, Topic, or Priority clears that stored
// value; omitted fields do not mean "leave unchanged". CreatedAt is immutable.
// This type does not validate input.
type ReplaceProblemParams struct {
	ProblemFields
	ProblemID int64
	ClassCode string
}

// PatchField distinguishes an omitted field from an explicitly supplied zero value.
// For nullable fields, use PatchField[*string]: Set=true with Value=nil clears it.
type PatchField[T any] struct {
	Set   bool
	Value T
}

// IsZero omits fields that were not supplied, regardless of their stored value.
func (field PatchField[T]) IsZero() bool {
	return !field.Set
}

// MarshalJSON encodes the supplied value directly, including explicit null.
func (field PatchField[T]) MarshalJSON() ([]byte, error) {
	return jsonv2.Marshal(field.Value)
}

// UnmarshalJSON records field presence while preserving JSON null as the zero value
// of T. In particular, PatchField[*string] distinguishes omission from null.
func (field *PatchField[T]) UnmarshalJSON(data []byte) error {
	field.Set = true
	return jsonv2.Unmarshal(data, &field.Value)
}

// PatchProblemFields contains only fields that may be changed by a patch.
type PatchProblemFields struct {
	ClassCode  PatchField[string]  `json:"class,omitzero"`
	Name       PatchField[string]  `json:"name,omitzero"`
	URL        PatchField[string]  `json:"url,omitzero"`
	Difficulty PatchField[*string] `json:"difficulty,omitzero"`
	Topic      PatchField[*string] `json:"topic,omitzero"`
	Priority   PatchField[*string] `json:"priority,omitzero"`
}

// PatchProblemParams identifies a problem and supplies only fields to change.
// An unset field is preserved; a set nullable field with a nil value is cleared.
type PatchProblemParams struct {
	PatchProblemFields
	ProblemID int64 `json:"problem_id"`
}

// ============================================================================
// Composite Models (Used for presentation/business logic)
// ============================================================================

// ProblemsByTopic represents a group of problems belonging to the same topic
type ProblemsByTopic struct {
	Topic    string
	Problems []Problem
}

// PairUserSolvedProblems represents a user and their solved problems
type PairUserSolvedProblems struct {
	User           User
	SolvedProblems []SolvedProblem
}

// PairUserProblemScore represents a user's score with problem completion status
type PairUserProblemScore struct {
	User          User
	CountProblems int
	Problems      map[int64]bool // Map problem ID to solved status
}

// Score represents the complete score information for a class
// Contains both user scores and problems grouped by topic
type Score struct {
	UserScores      []PairUserProblemScore
	ProblemsByTopic []ProblemsByTopic
}

// ClassScore pairs a class's stable identity and display name with its score.
// A slice of ClassScore values preserves classes that share a display name and
// supports a deterministic presentation order.
type ClassScore struct {
	ClassCode string
	ClassName string
	Score     Score
}

// Difficulty and priority codes are stable data, not display labels.
// An unassigned value is represented by nil in problem fields, not another code.
const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"

	PriorityEssential     = "essential"
	PriorityRecommended   = "recommended"
	PrioritySupplementary = "supplementary"
	PriorityNotRequired   = "not_required"
	PriorityDiscouraged   = "discouraged"
)

// ParseDifficulty normalizes input and validates it against canonical difficulty codes.
func ParseDifficulty(input string) (string, error) {
	return parseProblemCode(input, "difficulty", []string{
		DifficultyEasy, DifficultyMedium, DifficultyHard,
	})
}

// ParsePriority normalizes input and validates it against canonical priority codes.
func ParsePriority(input string) (string, error) {
	return parseProblemCode(input, "priority", []string{
		PriorityEssential, PriorityRecommended, PrioritySupplementary,
		PriorityNotRequired, PriorityDiscouraged,
	})
}

func parseProblemCode(input, field string, validCodes []string) (string, error) {
	code := strings.ToLower(strings.TrimSpace(input))
	for _, valid := range validCodes {
		if code == valid {
			return code, nil
		}
	}
	return "", fmt.Errorf("invalid %s. Valid options: %s", field, strings.Join(validCodes, ", "))
}

const (
	DefaultTopic = "General"
)

func (p *Problem) GetTopic() string {
	if p.Topic == nil {
		return DefaultTopic
	}
	return *p.Topic
}
