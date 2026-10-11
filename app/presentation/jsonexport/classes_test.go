package jsonexport

import (
	"encoding/json"
	"testing"

	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatClasses(t *testing.T) {
	t.Parallel()

	t.Run("empty classes list produces empty JSON array", func(t *testing.T) {
		t.Parallel()
		data, err := FormatClasses([]service_model.Class{})
		require.NoError(t, err)
		assert.Equal(t, "[]", string(data))
	})

	t.Run("sorts classes deterministically by classCode", func(t *testing.T) {
		t.Parallel()
		input := []service_model.Class{
			{ID: 2, ClassCode: "intermedios", Name: "Algoritmos Intermedios"},
			{ID: 1, ClassCode: "basicos", Name: "Algoritmos Básicos"},
			{ID: 3, ClassCode: "avanzados", Name: "Algoritmos Avanzados"},
		}

		data, err := FormatClasses(input)
		require.NoError(t, err)

		var output []JSONClassSummary
		err = json.Unmarshal(data, &output)
		require.NoError(t, err)

		require.Len(t, output, 3)
		assert.Equal(t, "avanzados", output[0].ClassCode)
		assert.Equal(t, "Algoritmos Avanzados", output[0].Name)
		assert.Equal(t, "basicos", output[1].ClassCode)
		assert.Equal(t, "Algoritmos Básicos", output[1].Name)
		assert.Equal(t, "intermedios", output[2].ClassCode)
		assert.Equal(t, "Algoritmos Intermedios", output[2].Name)
	})
}
