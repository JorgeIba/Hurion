package service_model

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
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

// NormalizeAndValidateReplaceProblemParams checks replacement values and identity
// without accessing storage. Nil optional values clear their stored values.
func NormalizeAndValidateReplaceProblemParams(params ReplaceProblemParams) (ReplaceProblemParams, error) {
	params.ClassCode = strings.TrimSpace(params.ClassCode)
	var err error
	params.ProblemFields, err = normalizeAndValidateProblemFields(params.ProblemFields)
	if params.ProblemID <= 0 {
		err = errors.Join(err, errors.New("problem ID must be positive"))
	}
	if params.ClassCode == "" {
		err = errors.Join(err, errors.New("class code is required"))
	}
	return params, errors.Join(err, ValidateClassCodeLength(params.ClassCode))
}

// NormalizeAndValidatePatchProblemParams validates only supplied values. Unset
// fields are left untouched; explicit nulls for nullable fields remain nil so they
// clear the stored value when applied.
func NormalizeAndValidatePatchProblemParams(params PatchProblemParams) (PatchProblemParams, error) {
	var problems []error
	if params.ProblemID <= 0 {
		problems = append(problems, errors.New("problem ID must be positive"))
	}
	if !params.ClassCode.Set && !params.Name.Set && !params.URL.Set && !params.Difficulty.Set && !params.Topic.Set && !params.Priority.Set {
		problems = append(problems, errors.New("at least one problem field must be provided"))
	}

	if params.ClassCode.Set {
		params.ClassCode.Value = strings.TrimSpace(params.ClassCode.Value)
		if params.ClassCode.Value == "" {
			problems = append(problems, errors.New("class code is required"))
		}
		problems = append(problems, ValidateClassCodeLength(params.ClassCode.Value))
	}
	if params.Name.Set {
		params.Name.Value = strings.TrimSpace(params.Name.Value)
		if params.Name.Value == "" {
			problems = append(problems, errors.New("problem name is required"))
		}
		problems = append(problems, validateTextLength("problem name", params.Name.Value, ProblemNameMaxLength))
	}
	if params.URL.Set {
		params.URL.Value = strings.TrimSpace(params.URL.Value)
		parsed, err := url.ParseRequestURI(params.URL.Value)
		if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			problems = append(problems, errors.New("URL must be an absolute HTTP or HTTPS URL"))
		}
		problems = append(problems, validateTextLength("URL", params.URL.Value, ProblemURLMaxLength))
	}
	if params.Topic.Set {
		params.Topic.Value = normalizeOptionalProblemText(params.Topic.Value)
		value := ""
		if params.Topic.Value != nil {
			value = *params.Topic.Value
		}
		problems = append(problems, validateTextLength("topic", value, ProblemTopicMaxLength))
	}
	if params.Difficulty.Set {
		params.Difficulty.Value = normalizeOptionalProblemText(params.Difficulty.Value)
		if params.Difficulty.Value != nil {
			code, err := ParseDifficulty(*params.Difficulty.Value)
			if err != nil {
				problems = append(problems, err)
			} else {
				params.Difficulty.Value = &code
			}
		}
	}
	if params.Priority.Set {
		params.Priority.Value = normalizeOptionalProblemText(params.Priority.Value)
		if params.Priority.Value != nil {
			code, err := ParsePriority(*params.Priority.Value)
			if err != nil {
				problems = append(problems, err)
			} else {
				params.Priority.Value = &code
			}
		}
	}
	return params, errors.Join(problems...)
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
