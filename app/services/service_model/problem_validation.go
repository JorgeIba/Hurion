package service_model

import (
	"errors"
	"net/url"
	"strings"
)

// NormalizeAndValidateCreateProblemParams checks problem values without looking up
// classes or checking permissions. It returns a copy and leaves supplied pointers unchanged.
func NormalizeAndValidateCreateProblemParams(params CreateProblemParams) (CreateProblemParams, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.URL = strings.TrimSpace(params.URL)
	params.ClassCode = strings.TrimSpace(params.ClassCode)

	var problems []error
	if params.Name == "" {
		problems = append(problems, errors.New("problem name is required"))
	}
	if params.ClassCode == "" {
		problems = append(problems, errors.New("class code is required"))
	}

	parsed, err := url.ParseRequestURI(params.URL)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		problems = append(problems, errors.New("URL must be an absolute HTTP or HTTPS URL"))
	}

	params.Topic = normalizeOptionalProblemText(params.Topic)
	params.Difficulty = normalizeOptionalProblemText(params.Difficulty)
	if params.Difficulty != nil {
		code, err := ParseDifficulty(*params.Difficulty)
		if err != nil {
			problems = append(problems, err)
		} else {
			params.Difficulty = &code
		}
	}

	params.Priority = normalizeOptionalProblemText(params.Priority)
	if params.Priority != nil {
		code, err := ParsePriority(*params.Priority)
		if err != nil {
			problems = append(problems, err)
		} else {
			params.Priority = &code
		}
	}
	return params, errors.Join(problems...)
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
