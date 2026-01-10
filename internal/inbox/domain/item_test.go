package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInboxMetadata_MarshalJSON(t *testing.T) {
	t.Run("marshals empty metadata", func(t *testing.T) {
		m := InboxMetadata{}

		data, err := m.MarshalJSON()

		require.NoError(t, err)
		assert.JSONEq(t, `{}`, string(data))
	})

	t.Run("marshals metadata with values", func(t *testing.T) {
		m := InboxMetadata{
			"source":   "email",
			"priority": "high",
		}

		data, err := m.MarshalJSON()

		require.NoError(t, err)

		var result map[string]string
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		assert.Equal(t, "email", result["source"])
		assert.Equal(t, "high", result["priority"])
	})

	t.Run("marshals nil metadata", func(t *testing.T) {
		var m InboxMetadata

		data, err := m.MarshalJSON()

		require.NoError(t, err)
		assert.JSONEq(t, `null`, string(data))
	})
}

func TestInboxMetadata_UnmarshalJSON(t *testing.T) {
	t.Run("unmarshals empty object", func(t *testing.T) {
		var m InboxMetadata
		err := m.UnmarshalJSON([]byte(`{}`))

		require.NoError(t, err)
		assert.Empty(t, m)
	})

	t.Run("unmarshals metadata with values", func(t *testing.T) {
		var m InboxMetadata
		err := m.UnmarshalJSON([]byte(`{"source":"cli","category":"work"}`))

		require.NoError(t, err)
		assert.Equal(t, "cli", m["source"])
		assert.Equal(t, "work", m["category"])
	})

	t.Run("fails on invalid JSON", func(t *testing.T) {
		var m InboxMetadata
		err := m.UnmarshalJSON([]byte(`{invalid}`))

		assert.Error(t, err)
	})

	t.Run("fails on non-object JSON", func(t *testing.T) {
		var m InboxMetadata
		err := m.UnmarshalJSON([]byte(`"string"`))

		assert.Error(t, err)
	})
}

func TestInboxMetadata_RoundTrip(t *testing.T) {
	t.Run("roundtrip preserves data", func(t *testing.T) {
		original := InboxMetadata{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var restored InboxMetadata
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.Equal(t, original, restored)
	})
}
