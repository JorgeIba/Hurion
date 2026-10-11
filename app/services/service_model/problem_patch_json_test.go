package service_model

import (
	jsonv2 "encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchProblemParamsJSONRoundTrip(t *testing.T) {
	t.Parallel()
	for _, input := range []string{
		`{"problem_id":42}`,
		`{"problem_id":42,"class":"algorithms","name":"Two Sum","url":"https://example.com/problem","difficulty":"easy","topic":"graphs","priority":"high"}`,
		`{"problem_id":42,"difficulty":null,"topic":null,"priority":null}`,
		`{"problem_id":42,"class":"","name":"","url":"","difficulty":"","topic":"","priority":""}`,
	} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			var original PatchProblemParams
			require.NoError(t, jsonv2.Unmarshal([]byte(input), &original))

			encoded, err := jsonv2.Marshal(original)
			require.NoError(t, err)
			assert.JSONEq(t, input, string(encoded))

			var decoded PatchProblemParams
			require.NoError(t, jsonv2.Unmarshal(encoded, &decoded))
			assert.Equal(t, original, decoded)
		})
	}
}

func TestPatchProblemParamsJSONOmitsUnsetFieldsWithStoredValues(t *testing.T) {
	t.Parallel()
	topic := "graphs"
	params := PatchProblemParams{
		ProblemID: 42,
		PatchProblemFields: PatchProblemFields{
			Name:  PatchField[string]{Value: "Unused name"},
			Topic: PatchField[*string]{Value: &topic},
		},
	}

	encoded, err := jsonv2.Marshal(params)
	require.NoError(t, err)
	assert.JSONEq(t, `{"problem_id":42}`, string(encoded))
}

func TestPatchProblemParamsJSONPreservesEachFieldIntent(t *testing.T) {
	t.Parallel()
	fields := []struct {
		name     string
		nullable bool
		make     func(bool, string, bool) PatchProblemFields
	}{
		{"class", false, func(set bool, value string, _ bool) PatchProblemFields {
			return PatchProblemFields{ClassCode: PatchField[string]{Set: set, Value: value}}
		}},
		{"name", false, func(set bool, value string, _ bool) PatchProblemFields {
			return PatchProblemFields{Name: PatchField[string]{Set: set, Value: value}}
		}},
		{"url", false, func(set bool, value string, _ bool) PatchProblemFields {
			return PatchProblemFields{URL: PatchField[string]{Set: set, Value: value}}
		}},
		{"difficulty", true, func(set bool, value string, clear bool) PatchProblemFields {
			var pointer *string
			if !clear {
				pointer = &value
			}
			return PatchProblemFields{Difficulty: PatchField[*string]{Set: set, Value: pointer}}
		}},
		{"topic", true, func(set bool, value string, clear bool) PatchProblemFields {
			var pointer *string
			if !clear {
				pointer = &value
			}
			return PatchProblemFields{Topic: PatchField[*string]{Set: set, Value: pointer}}
		}},
		{"priority", true, func(set bool, value string, clear bool) PatchProblemFields {
			var pointer *string
			if !clear {
				pointer = &value
			}
			return PatchProblemFields{Priority: PatchField[*string]{Set: set, Value: pointer}}
		}},
	}
	for _, field := range fields {
		for _, state := range []struct {
			name  string
			set   bool
			value string
			clear bool
		}{
			{name: "omitted"},
			{name: "omitted with stored value", value: "stale value"},
			{name: "explicit value", set: true, value: "Árboles \"y\" grafos\n第二"},
			{name: "explicit empty string", set: true},
			{name: "explicit null", set: true, clear: true},
		} {
			if state.clear && !field.nullable {
				continue
			}
			t.Run(field.name+"/"+state.name, func(t *testing.T) {
				t.Parallel()
				params := PatchProblemParams{ProblemID: 42, PatchProblemFields: field.make(state.set, state.value, state.clear)}
				expected := map[string]any{"problem_id": 42}
				if state.set {
					if state.clear {
						expected[field.name] = nil
					} else {
						expected[field.name] = state.value
					}
				}
				expectedJSON, err := jsonv2.Marshal(expected)
				require.NoError(t, err)
				encoded, err := jsonv2.Marshal(params)
				require.NoError(t, err)
				assert.JSONEq(t, string(expectedJSON), string(encoded))

				var decoded PatchProblemParams
				require.NoError(t, jsonv2.Unmarshal(encoded, &decoded))
				if !state.set {
					params.PatchProblemFields = PatchProblemFields{}
				}
				assert.Equal(t, params, decoded)
			})
		}
	}
}

func TestPatchProblemParamsJSONBatchPreservesIndependentUpdates(t *testing.T) {
	t.Parallel()
	difficulty, topic := "hard", "graphs"
	params := []PatchProblemParams{
		{ProblemID: 1, PatchProblemFields: PatchProblemFields{Difficulty: PatchField[*string]{Set: true, Value: &difficulty}}},
		{ProblemID: 2, PatchProblemFields: PatchProblemFields{Topic: PatchField[*string]{Set: true}}},
		{ProblemID: 3, PatchProblemFields: PatchProblemFields{Topic: PatchField[*string]{Set: true, Value: &topic}, Priority: PatchField[*string]{Set: true}}},
		{ProblemID: 4},
	}
	encoded, err := jsonv2.Marshal(params)
	require.NoError(t, err)
	assert.JSONEq(t, `[{"problem_id":1,"difficulty":"hard"},{"problem_id":2,"topic":null},{"problem_id":3,"topic":"graphs","priority":null},{"problem_id":4}]`, string(encoded))

	var decoded []PatchProblemParams
	require.NoError(t, jsonv2.Unmarshal(encoded, &decoded))
	assert.Equal(t, params, decoded)
}
