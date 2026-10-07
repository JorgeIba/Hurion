package service_model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAndValidateCreateProblemParamsNormalizesCopyWithoutChangingInput(t *testing.T) {
	difficulty, priority, topic := " MEDIUM ", " RECOMMENDED ", " Binary search "
	input := CreateProblemParams{
		Name: " Two Sum 🚀 ", URL: " https://example.com/problems?q=two ", ClassCode: " basicos_1 ",
		Difficulty: &difficulty, Priority: &priority, Topic: &topic,
	}
	result, err := NormalizeAndValidateCreateProblemParams(input)
	require.NoError(t, err)
	assert.Equal(t, "Two Sum 🚀", result.Name)
	assert.Equal(t, "https://example.com/problems?q=two", result.URL)
	assert.Equal(t, "basicos_1", result.ClassCode)
	assert.Equal(t, "medium", *result.Difficulty)
	assert.Equal(t, "recommended", *result.Priority)
	assert.Equal(t, "Binary search", *result.Topic)
	assert.Equal(t, " Two Sum 🚀 ", input.Name)
	assert.Equal(t, " MEDIUM ", difficulty)
	assert.Equal(t, " RECOMMENDED ", priority)
	assert.Equal(t, " Binary search ", topic)
}

func TestNormalizeAndValidateCreateProblemParamsRejectsInvalidFields(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*CreateProblemParams)
		want   string
	}{
		{"blank name", func(p *CreateProblemParams) { p.Name = " \t" }, "problem name is required"},
		{"blank class", func(p *CreateProblemParams) { p.ClassCode = " " }, "class code is required"},
		{"blank URL", func(p *CreateProblemParams) { p.URL = "" }, "absolute HTTP or HTTPS"},
		{"relative URL", func(p *CreateProblemParams) { p.URL = "/problem/1" }, "absolute HTTP or HTTPS"},
		{"missing host", func(p *CreateProblemParams) { p.URL = "https:///problem" }, "absolute HTTP or HTTPS"},
		{"unsupported scheme", func(p *CreateProblemParams) { p.URL = "javascript:alert(1)" }, "absolute HTTP or HTTPS"},
		{"malformed URL", func(p *CreateProblemParams) { p.URL = "https://example.com/%zz" }, "absolute HTTP or HTTPS"},
		{"invalid difficulty", func(p *CreateProblemParams) { v := "extreme"; p.Difficulty = &v }, "invalid difficulty"},
		{"invalid priority", func(p *CreateProblemParams) { v := "urgent"; p.Priority = &v }, "invalid priority"},
	} {
		t.Run(test.name, func(t *testing.T) {
			params := CreateProblemParams{Name: "Two Sum", URL: "https://example.com/problem", ClassCode: "basicos_1"}
			test.change(&params)
			_, err := NormalizeAndValidateCreateProblemParams(params)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestNormalizeAndValidateCreateProblemParamsAcceptsMissingAndBlankOptionalValues(t *testing.T) {
	empty, whitespace := "", " \t"
	for _, test := range []struct {
		name  string
		value *string
	}{
		{name: "missing", value: nil},
		{name: "empty", value: &empty},
		{name: "whitespace", value: &whitespace},
	} {
		t.Run(test.name, func(t *testing.T) {
			params := CreateProblemParams{Name: "Two Sum", URL: "http://example.com/problem", ClassCode: "basicos_1", Difficulty: test.value, Priority: test.value, Topic: test.value}
			result, err := NormalizeAndValidateCreateProblemParams(params)
			require.NoError(t, err)
			assert.Nil(t, result.Difficulty)
			assert.Nil(t, result.Priority)
			assert.Nil(t, result.Topic)
		})
	}
}

func TestNormalizeAndValidateCreateProblemParamsReportsAllInvalidFields(t *testing.T) {
	_, err := NormalizeAndValidateCreateProblemParams(CreateProblemParams{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "problem name is required")
	assert.ErrorContains(t, err, "class code is required")
	assert.ErrorContains(t, err, "URL must be")
}

func TestNormalizeAndValidateUpdateProblemParamsPreservesIdentityAndInput(t *testing.T) {
	difficulty, topic, priority := " HARD ", " Graphs ", " ESSENTIAL "
	input := UpdateProblemParams{
		ProblemID: 42, ClassID: 9, Name: " Two Sum ", URL: " https://example.com/problem ",
		Difficulty: &difficulty, Topic: &topic, Priority: &priority,
	}
	result, err := NormalizeAndValidateUpdateProblemParams(input)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, int64(42), result.ProblemID)
	assert.Equal(t, int64(9), result.ClassID)
	assert.Equal(t, "Two Sum", result.Name)
	assert.Equal(t, "https://example.com/problem", result.URL)
	assert.Equal(t, "hard", *result.Difficulty)
	assert.Equal(t, "Graphs", *result.Topic)
	assert.Equal(t, "essential", *result.Priority)
	assert.Equal(t, " Two Sum ", input.Name)
	assert.Equal(t, " https://example.com/problem ", input.URL)
	assert.Equal(t, " HARD ", difficulty)
	assert.Equal(t, " Graphs ", topic)
	assert.Equal(t, " ESSENTIAL ", priority)
}

func TestNormalizeAndValidateUpdateProblemParamsNormalizesClearedValues(t *testing.T) {
	empty, whitespace := "", " \t"
	for _, tt := range []struct {
		name  string
		value *string
	}{
		{name: "nil", value: nil},
		{name: "empty", value: &empty},
		{name: "whitespace", value: &whitespace},
	} {
		t.Run(tt.name, func(t *testing.T) {
			params := UpdateProblemParams{
				ProblemID: 42, ClassID: 9, Name: "Two Sum", URL: "https://example.com/problem",
				Difficulty: tt.value, Topic: tt.value, Priority: tt.value,
			}
			result, err := NormalizeAndValidateUpdateProblemParams(params)
			assert.NoError(t, err)
			assert.Nil(t, result.Difficulty)
			assert.Nil(t, result.Topic)
			assert.Nil(t, result.Priority)
		})
	}
}
