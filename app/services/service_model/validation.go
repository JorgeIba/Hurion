package service_model

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

// Text limits count Unicode code points, not UTF-8 bytes.
const (
	ProblemNameMaxLength  = 200
	ProblemURLMaxLength   = 2000
	ProblemTopicMaxLength = 100
	ClassCodeMaxLength    = 50
)

// NormalizeAndValidateCreateProblemParams checks problem values without looking up
// classes or checking permissions. It returns a copy and leaves supplied pointers unchanged.
func NormalizeAndValidateCreateProblemParams(params CreateProblemParams) (CreateProblemParams, error) {
	params.ClassCode = strings.TrimSpace(params.ClassCode)
	var err error
	params.ProblemFields, err = normalizeAndValidateProblemFields(params.ProblemFields)
	if params.ClassCode == "" {
		err = errors.Join(err, errors.New("class code is required"))
	}
	return params, errors.Join(err, ValidateClassCodeLength(params.ClassCode))
}

// NormalizeAndValidateUpdateProblemParams checks replacement values and identity
// without accessing storage. Nil optional values clear their stored values.
func NormalizeAndValidateUpdateProblemParams(params UpdateProblemParams) (UpdateProblemParams, error) {
	var err error
	params.ProblemFields, err = normalizeAndValidateProblemFields(params.ProblemFields)
	if params.ProblemID <= 0 {
		err = errors.Join(err, errors.New("problem ID must be positive"))
	}
	if params.ClassID <= 0 {
		err = errors.Join(err, errors.New("destination class ID must be positive"))
	}
	return params, err
}

func normalizeAndValidateProblemFields(fields ProblemFields) (ProblemFields, error) {
	fields.Name = strings.TrimSpace(fields.Name)
	fields.URL = strings.TrimSpace(fields.URL)

	var problems []error
	if fields.Name == "" {
		problems = append(problems, errors.New("problem name is required"))
	}

	parsed, err := url.ParseRequestURI(fields.URL)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		problems = append(problems, errors.New("URL must be an absolute HTTP or HTTPS URL"))
	}

	fields.Topic = normalizeOptionalProblemText(fields.Topic)
	problems = append(problems, ValidateProblemTextLengths(fields.Name, fields.URL, fields.Topic))
	fields.Difficulty = normalizeOptionalProblemText(fields.Difficulty)
	if fields.Difficulty != nil {
		code, err := ParseDifficulty(*fields.Difficulty)
		if err != nil {
			problems = append(problems, err)
		} else {
			fields.Difficulty = &code
		}
	}

	fields.Priority = normalizeOptionalProblemText(fields.Priority)
	if fields.Priority != nil {
		code, err := ParsePriority(*fields.Priority)
		if err != nil {
			problems = append(problems, err)
		} else {
			fields.Priority = &code
		}
	}
	return fields, errors.Join(problems...)
}

// ValidateProblemTextLengths checks creation and replacement values without
// normalization, lookups, or required-field checks.
func ValidateProblemTextLengths(name, problemURL string, topic *string) error {
	topicValue := ""
	if topic != nil {
		topicValue = *topic
	}
	return errors.Join(
		validateTextLength("problem name", name, ProblemNameMaxLength),
		validateTextLength("URL", problemURL, ProblemURLMaxLength),
		validateTextLength("topic", topicValue, ProblemTopicMaxLength),
	)
}

// ValidateClassCodeLength applies the same limit to class creation and selection.
func ValidateClassCodeLength(classCode string) error {
	return validateTextLength("class code", classCode, ClassCodeMaxLength)
}

// validateTextLength counts Unicode code points rather than UTF-8 bytes.
func validateTextLength(field, value string, limit int) error {
	if utf8.RuneCountInString(value) > limit {
		return fmt.Errorf("%s must be at most %d characters", field, limit)
	}
	return nil
}

func normalizeOptionalProblemText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
