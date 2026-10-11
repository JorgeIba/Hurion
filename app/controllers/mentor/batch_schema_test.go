package mentor

import (
	jsonv2 "encoding/json/v2"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/JorgeIba/Hurion/app/batchschema"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type batchSchemaProperty struct {
	Type      any    `json:"type"`
	MinLength int    `json:"minLength"`
	MaxLength int    `json:"maxLength"`
	Minimum   *int64 `json:"minimum"`
	Maximum   *int64 `json:"maximum"`
	Pattern   string `json:"pattern"`
	Format    string `json:"format"`
	Enum      []any  `json:"enum"`
	Not       *struct {
		Type    string `json:"type"`
		Pattern string `json:"pattern"`
	} `json:"not"`
}

// batchSchema describes the JSON Schema keywords used by these batch contracts.
type batchSchema struct {
	Type     string `json:"type"`
	MinItems int    `json:"minItems"`
	MaxItems int    `json:"maxItems"`
	Items    struct {
		Type     string   `json:"type"`
		Required []string `json:"required"`
		AnyOf    []struct {
			Required []string `json:"required"`
		} `json:"anyOf"`
		AdditionalProperties *bool                          `json:"additionalProperties"`
		Properties           map[string]batchSchemaProperty `json:"properties"`
	} `json:"items"`
	Examples [][]map[string]any `json:"examples"`
}

// checkBatch runs the same parsing and validation as a submitted batch form.
func checkBatch[T any](validateEntry func(T) error) func(string) []batchInputIssue {
	return func(input string) []batchInputIssue {
		document, err := parseJSONDocument(input)
		if err != nil {
			return []batchInputIssue{{err: err}}
		}
		entries, issues := parseBatchInput[T](document)
		if len(issues) > 0 {
			return issues
		}
		return entries.validate(validateEntry)
	}
}

func batchInputJSONFields[T any](t *testing.T, example map[string]any) []string {
	t.Helper()
	encoded, err := jsonv2.Marshal(example)
	require.NoError(t, err)
	var input T
	require.NoError(t, jsonv2.Unmarshal(encoded, &input))
	encoded, err = jsonv2.Marshal(input)
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, jsonv2.Unmarshal(encoded, &fields))
	return slices.Collect(maps.Keys(fields))
}

// These tests cover the schema keywords and boundaries used by the published
// contracts without adding a JSON Schema engine to the production bot.
func TestBatchSchemasMatchBatchValidation(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		file        string
		inputFields func(*testing.T, map[string]any) []string
		check       func(string) []batchInputIssue
	}{
		{"problem-add-batch.json", batchInputJSONFields[problemInput], checkBatch(validateProblemInput)},
		{"problem-edit-batch.json", batchInputJSONFields[problemEditInput], checkBatch(validateProblemEditInput)},
		{"ac-add-batch.json", batchInputJSONFields[acInput], checkBatch(validateACInput)},
	} {
		t.Run(test.file, func(t *testing.T) {
			t.Parallel()
			raw, err := batchschema.FS.ReadFile(test.file)
			require.NoError(t, err)
			var schema batchSchema
			require.NoError(t, jsonv2.Unmarshal(raw, &schema))
			require.NotEmpty(t, schema.Examples)
			require.NotEmpty(t, schema.Examples[0])
			require.Equal(t, "array", schema.Type)
			require.Equal(t, "object", schema.Items.Type)
			require.NotNil(t, schema.Items.AdditionalProperties)
			assert.False(t, *schema.Items.AdditionalProperties)
			assert.Equal(t, 1, schema.MinItems)
			assert.Equal(t, service_model.MaxBatchOperations, schema.MaxItems)

			batch := func(entries ...map[string]any) string {
				document, err := jsonv2.Marshal(entries)
				require.NoError(t, err)
				return string(document)
			}
			entry := func() map[string]any { return maps.Clone(schema.Examples[0][0]) }

			t.Run("properties are the accepted fields", func(t *testing.T) {
				assert.ElementsMatch(t, test.inputFields(t, schema.Examples[0][0]), slices.Collect(maps.Keys(schema.Items.Properties)))
			})

			if test.file == "problem-edit-batch.json" {
				t.Run("requires a target and at least one patch field", func(t *testing.T) {
					assert.Equal(t, []string{"problem_id"}, schema.Items.Required)
					patchFields := make([]string, len(schema.Items.AnyOf))
					for index, alternative := range schema.Items.AnyOf {
						require.Len(t, alternative.Required, 1)
						patchFields[index] = alternative.Required[0]
					}
					assert.ElementsMatch(t, []string{"class", "name", "url", "difficulty", "topic", "priority"}, patchFields)

					assert.NotEmpty(t, test.check(batch(map[string]any{"problem_id": int64(42)})), "problem_id alone must not be a valid edit")
					assert.Empty(t, test.check(batch(map[string]any{"problem_id": int64(42), "name": "Updated"})), "one valid patch field must be enough")
				})
			}

			t.Run("unknown fields are rejected", func(t *testing.T) {
				value := entry()
				value["unexpected_field"] = true
				assert.NotEmpty(t, test.check(batch(value)))
			})

			t.Run("examples are accepted", func(t *testing.T) {
				for _, example := range schema.Examples {
					assert.Empty(t, test.check(batch(example...)))
				}
			})

			t.Run("batch size limits", func(t *testing.T) {
				sized := func(n int) string {
					entries := make([]map[string]any, n)
					for i := range entries {
						entries[i] = entry()
					}
					return batch(entries...)
				}
				assert.NotEmpty(t, test.check(sized(schema.MinItems-1)))
				assert.Empty(t, test.check(sized(schema.MinItems)))
				assert.Empty(t, test.check(sized(schema.MaxItems)))
				assert.NotEmpty(t, test.check(sized(schema.MaxItems+1)))
			})

			for name, property := range schema.Items.Properties {
				t.Run(name+" type", func(t *testing.T) {
					wantType := any("string")
					switch name {
					case "problem_id":
						wantType = "integer"
					case "topic":
						if test.file != "problem-add-batch.json" {
							wantType = []any{"string", "null"}
						}
					case "difficulty", "priority":
						wantType = []any{"string", "null"}
					}
					assert.Equal(t, wantType, property.Type)

					if name == "difficulty" || name == "priority" || name == "topic" {
						value := entry()
						value[name] = nil
						if name == "topic" && test.file == "problem-add-batch.json" {
							assert.NotEmpty(t, test.check(batch(value)), "creation requires a non-null topic")
						} else {
							assert.Empty(t, test.check(batch(value)), "bot must accept null where the schema declares it")
						}
					}

					wrongType := any(42)
					if name == "problem_id" {
						wrongType = "42"
					}
					value := entry()
					value[name] = wrongType
					assert.NotEmpty(t, test.check(batch(value)), "bot must reject a type outside the schema")

					if name == "url" {
						assert.Equal(t, "uri", property.Format)
					}
				})

				switch name {
				case "class":
					assert.Equal(t, service_model.ClassCodeMaxLength, property.MaxLength)
					assert.Equal(t, 1, property.MinLength)
				case "name":
					assert.Equal(t, service_model.ProblemNameMaxLength, property.MaxLength)
					assert.Equal(t, 1, property.MinLength)
				case "url":
					assert.Equal(t, service_model.ProblemURLMaxLength, property.MaxLength)
					assert.Equal(t, 1, property.MinLength)
					assert.Equal(t, "uri", property.Format)
				case "topic":
					assert.Equal(t, service_model.ProblemTopicMaxLength, property.MaxLength)
					assert.Equal(t, test.file == "problem-add-batch.json", slices.Contains(schema.Items.Required, name))
					require.NotNil(t, property.Not)
					assert.Equal(t, "string", property.Not.Type, "the exclusion must not reject null on a nullable edit")
					pattern, err := regexp.Compile(property.Not.Pattern)
					require.NoError(t, err)
					for _, testCase := range []struct {
						value string
						valid bool
					}{
						{value: "General"},
						{value: " general "},
						{value: "Dynamic Programming", valid: true},
					} {
						value := entry()
						value[name] = testCase.value
						assert.Equal(t, testCase.valid, !pattern.MatchString(testCase.value), "schema rule for topic %q", testCase.value)
						assert.Equal(t, testCase.valid, len(test.check(batch(value))) == 0, "Go validation for topic %q", testCase.value)
					}
				case "problem_id":
					require.NotNil(t, property.Minimum)
					require.NotNil(t, property.Maximum)
					assert.Equal(t, int64(1), *property.Minimum)
				case "user_id":
					assert.Equal(t, 20, property.MaxLength)
				}

				t.Run(name+" required", func(t *testing.T) {
					value := entry()
					delete(value, name)
					issues := test.check(batch(value))
					if slices.Contains(schema.Items.Required, name) {
						assert.NotEmpty(t, issues, "omitting a required field must be rejected")
					} else {
						assert.Empty(t, issues, "omitting an optional field must be accepted")
					}
				})

				if property.MinLength > 0 {
					t.Run(name+" minimum length", func(t *testing.T) {
						value := entry()
						value[name] = ""
						assert.NotEmpty(t, test.check(batch(value)))
					})
				}

				if property.MaxLength > 0 {
					t.Run(name+" maxLength", func(t *testing.T) {
						base, ok := schema.Examples[0][0][name].(string)
						require.True(t, ok, "example must provide a string to pad")
						pad := func(n int) map[string]any {
							value := entry()
							padding := "a"
							if name == "user_id" {
								padding = "1"
							}
							value[name] = base + strings.Repeat(padding, n-utf8.RuneCountInString(base))
							return value
						}
						assert.Empty(t, test.check(batch(pad(property.MaxLength))))
						assert.NotEmpty(t, test.check(batch(pad(property.MaxLength+1))))
					})
				}

				if property.Minimum != nil {
					t.Run(name+" numeric minimum", func(t *testing.T) {
						assert.Equal(t, int64(1), *property.Minimum)
						value := entry()
						value[name] = *property.Minimum
						assert.Empty(t, test.check(batch(value)))
						value[name] = int64(0)
						assert.NotEmpty(t, test.check(batch(value)))
					})
				}

				if property.Maximum != nil {
					t.Run(name+" numeric maximum", func(t *testing.T) {
						const maxInt64 = int64(1<<63 - 1)
						require.Equal(t, maxInt64, *property.Maximum)
						value := entry()
						value[name] = *property.Maximum
						assert.Empty(t, test.check(batch(value)))

						encoded := batch(value)
						maximum := `"` + name + `":` + strconv.FormatInt(*property.Maximum, 10)
						tooLarge := `"` + name + `":9223372036854775808`
						overflow := strings.Replace(encoded, maximum, tooLarge, 1)
						require.NotEqual(t, encoded, overflow, "test input must contain the maximum value")
						assert.NotEmpty(t, test.check(overflow))
					})
				}

				if len(property.Enum) > 0 {
					t.Run(name+" enum", func(t *testing.T) {
						for _, allowed := range property.Enum {
							value := entry()
							value[name] = allowed
							assert.Empty(t, test.check(batch(value)), "schema allows %v", allowed)
						}
						value := entry()
						value[name] = "not_a_listed_value"
						assert.NotEmpty(t, test.check(batch(value)))
					})
				}

				if property.Pattern != "" {
					t.Run(name+" pattern accepts its example", func(t *testing.T) {
						pattern, err := regexp.Compile(property.Pattern)
						require.NoError(t, err)
						value, ok := schema.Examples[0][0][name].(string)
						require.True(t, ok, "pattern applies to a string example")
						assert.True(t, pattern.MatchString(value))
					})
				}
			}

			t.Run("schema patterns agree with Go validation", func(t *testing.T) {
				cases := []struct {
					field string
					value string
					valid bool
				}{
					{field: "class", value: "   ", valid: false},
					{field: "name", value: "   ", valid: false},
					{field: "url", value: "https:///problem", valid: false},
					{field: "url", value: "https://:80", valid: false},
					{field: "url", value: "https://example.com:bad/", valid: false},
					{field: "url", value: "https://example.com/problem", valid: true},
					{field: "url", value: "https://example.com:443/problem", valid: true},
					{field: "user_id", value: "18446744073709551615", valid: true},
					{field: "user_id", value: "18446744073709551614", valid: true},
					{field: "user_id", value: "18446744073709551616", valid: false},
				}
				for _, testCase := range cases {
					property, ok := schema.Items.Properties[testCase.field]
					if !ok {
						continue
					}
					require.NotEmpty(t, property.Pattern)
					pattern, err := regexp.Compile(property.Pattern)
					require.NoError(t, err)
					value := entry()
					value[testCase.field] = testCase.value
					assert.Equal(t, testCase.valid, pattern.MatchString(testCase.value), "%s pattern for %q", testCase.field, testCase.value)
					assert.Equal(t, testCase.valid, len(test.check(batch(value))) == 0, "Go validation for %s=%q", testCase.field, testCase.value)
				}
			})
		})
	}
}
