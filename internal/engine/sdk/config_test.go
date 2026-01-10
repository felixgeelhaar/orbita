package sdk

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngineConfig(t *testing.T) {
	t.Run("creates config with all fields", func(t *testing.T) {
		engineID := "acme.scheduler"
		userID := uuid.New()
		raw := map[string]any{"key": "value"}

		config := NewEngineConfig(engineID, userID, raw)

		assert.Equal(t, engineID, config.EngineID)
		assert.Equal(t, userID, config.UserID)
		assert.Equal(t, raw, config.Raw)
	})

	t.Run("creates config with nil raw", func(t *testing.T) {
		config := NewEngineConfig("test", uuid.New(), nil)

		assert.NotNil(t, config.Raw)
		assert.Empty(t, config.Raw)
	})
}

func TestEngineConfig_Get(t *testing.T) {
	config := NewEngineConfig("test", uuid.New(), map[string]any{
		"string": "value",
		"number": 42,
		"nil":    nil,
	})

	t.Run("returns existing value", func(t *testing.T) {
		assert.Equal(t, "value", config.Get("string"))
		assert.Equal(t, 42, config.Get("number"))
	})

	t.Run("returns nil for missing key", func(t *testing.T) {
		assert.Nil(t, config.Get("missing"))
	})

	t.Run("returns nil for nil value", func(t *testing.T) {
		assert.Nil(t, config.Get("nil"))
	})
}

func TestEngineConfig_GetString(t *testing.T) {
	config := NewEngineConfig("test", uuid.New(), map[string]any{
		"string": "hello",
		"number": 42,
		"empty":  "",
	})

	t.Run("returns string value", func(t *testing.T) {
		assert.Equal(t, "hello", config.GetString("string"))
	})

	t.Run("returns empty for non-string", func(t *testing.T) {
		assert.Equal(t, "", config.GetString("number"))
	})

	t.Run("returns empty for missing", func(t *testing.T) {
		assert.Equal(t, "", config.GetString("missing"))
	})

	t.Run("returns empty string value", func(t *testing.T) {
		assert.Equal(t, "", config.GetString("empty"))
	})
}

func TestEngineConfig_GetInt(t *testing.T) {
	config := NewEngineConfig("test", uuid.New(), map[string]any{
		"int":     42,
		"int64":   int64(100),
		"float64": 3.7,
		"json":    json.Number("999"),
		"string":  "42",
	})

	t.Run("returns int value", func(t *testing.T) {
		assert.Equal(t, 42, config.GetInt("int"))
	})

	t.Run("converts int64 to int", func(t *testing.T) {
		assert.Equal(t, 100, config.GetInt("int64"))
	})

	t.Run("converts float64 to int (truncates)", func(t *testing.T) {
		assert.Equal(t, 3, config.GetInt("float64"))
	})

	t.Run("converts json.Number to int", func(t *testing.T) {
		assert.Equal(t, 999, config.GetInt("json"))
	})

	t.Run("returns 0 for non-numeric", func(t *testing.T) {
		assert.Equal(t, 0, config.GetInt("string"))
	})

	t.Run("returns 0 for missing", func(t *testing.T) {
		assert.Equal(t, 0, config.GetInt("missing"))
	})
}

func TestEngineConfig_GetFloat(t *testing.T) {
	config := NewEngineConfig("test", uuid.New(), map[string]any{
		"float64": 3.14,
		"float32": float32(2.5),
		"int":     42,
		"int64":   int64(100),
		"json":    json.Number("1.5"),
		"string":  "3.14",
	})

	t.Run("returns float64 value", func(t *testing.T) {
		assert.Equal(t, 3.14, config.GetFloat("float64"))
	})

	t.Run("converts float32 to float64", func(t *testing.T) {
		assert.InDelta(t, 2.5, config.GetFloat("float32"), 0.001)
	})

	t.Run("converts int to float64", func(t *testing.T) {
		assert.Equal(t, 42.0, config.GetFloat("int"))
	})

	t.Run("converts int64 to float64", func(t *testing.T) {
		assert.Equal(t, 100.0, config.GetFloat("int64"))
	})

	t.Run("converts json.Number to float64", func(t *testing.T) {
		assert.Equal(t, 1.5, config.GetFloat("json"))
	})

	t.Run("returns 0 for non-numeric", func(t *testing.T) {
		assert.Equal(t, 0.0, config.GetFloat("string"))
	})

	t.Run("returns 0 for missing", func(t *testing.T) {
		assert.Equal(t, 0.0, config.GetFloat("missing"))
	})
}

func TestEngineConfig_GetBool(t *testing.T) {
	config := NewEngineConfig("test", uuid.New(), map[string]any{
		"true":   true,
		"false":  false,
		"string": "true",
	})

	t.Run("returns true value", func(t *testing.T) {
		assert.True(t, config.GetBool("true"))
	})

	t.Run("returns false value", func(t *testing.T) {
		assert.False(t, config.GetBool("false"))
	})

	t.Run("returns false for non-bool", func(t *testing.T) {
		assert.False(t, config.GetBool("string"))
	})

	t.Run("returns false for missing", func(t *testing.T) {
		assert.False(t, config.GetBool("missing"))
	})
}

func TestEngineConfig_GetDuration(t *testing.T) {
	config := NewEngineConfig("test", uuid.New(), map[string]any{
		"duration": "5s",
		"complex":  "1h30m",
		"invalid":  "not-a-duration",
		"number":   42,
	})

	t.Run("returns parsed duration", func(t *testing.T) {
		assert.Equal(t, 5*time.Second, config.GetDuration("duration"))
	})

	t.Run("parses complex duration", func(t *testing.T) {
		assert.Equal(t, 90*time.Minute, config.GetDuration("complex"))
	})

	t.Run("returns 0 for invalid duration string", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), config.GetDuration("invalid"))
	})

	t.Run("returns 0 for non-string", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), config.GetDuration("number"))
	})

	t.Run("returns 0 for missing", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), config.GetDuration("missing"))
	})
}

func TestEngineConfig_GetStringSlice(t *testing.T) {
	config := NewEngineConfig("test", uuid.New(), map[string]any{
		"strings": []any{"a", "b", "c"},
		"mixed":   []any{"a", 1, "b"},
		"empty":   []any{},
		"string":  "not-a-slice",
	})

	t.Run("returns string slice", func(t *testing.T) {
		result := config.GetStringSlice("strings")
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("filters non-strings from mixed slice", func(t *testing.T) {
		result := config.GetStringSlice("mixed")
		assert.Equal(t, []string{"a", "b"}, result)
	})

	t.Run("returns empty slice for empty array", func(t *testing.T) {
		result := config.GetStringSlice("empty")
		assert.Empty(t, result)
	})

	t.Run("returns nil for non-array", func(t *testing.T) {
		result := config.GetStringSlice("string")
		assert.Nil(t, result)
	})

	t.Run("returns nil for missing", func(t *testing.T) {
		result := config.GetStringSlice("missing")
		assert.Nil(t, result)
	})
}

func TestEngineConfig_Has(t *testing.T) {
	config := NewEngineConfig("test", uuid.New(), map[string]any{
		"key": "value",
		"nil": nil,
	})

	t.Run("returns true for existing key", func(t *testing.T) {
		assert.True(t, config.Has("key"))
	})

	t.Run("returns true for nil value", func(t *testing.T) {
		assert.True(t, config.Has("nil"))
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		assert.False(t, config.Has("missing"))
	})
}

func TestEngineConfig_Merge(t *testing.T) {
	t.Run("merges two configs", func(t *testing.T) {
		userID := uuid.New()
		c1 := NewEngineConfig("test", userID, map[string]any{"a": 1, "b": 2})
		c2 := NewEngineConfig("other", uuid.New(), map[string]any{"b": 3, "c": 4})

		merged := c1.Merge(c2)

		assert.Equal(t, 1, merged.Get("a"))
		assert.Equal(t, 3, merged.Get("b")) // c2 takes precedence
		assert.Equal(t, 4, merged.Get("c"))
		assert.Equal(t, userID, merged.UserID)       // Preserves c1's userID
		assert.Equal(t, "test", merged.EngineID)    // Preserves c1's engineID
	})

	t.Run("handles empty configs", func(t *testing.T) {
		c1 := NewEngineConfig("test", uuid.New(), map[string]any{"a": 1})
		c2 := NewEngineConfig("test", uuid.New(), nil)

		merged := c1.Merge(c2)

		assert.Equal(t, 1, merged.Get("a"))
	})
}

func TestNewConfigSchema(t *testing.T) {
	t.Run("creates schema with defaults", func(t *testing.T) {
		schema := NewConfigSchema("Test Config", "A test configuration")

		assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", schema.Schema)
		assert.Equal(t, "object", schema.Type)
		assert.Equal(t, "Test Config", schema.Title)
		assert.Equal(t, "A test configuration", schema.Description)
		assert.NotNil(t, schema.Properties)
		assert.Empty(t, schema.Properties)
	})
}

func TestConfigSchema_AddProperty(t *testing.T) {
	t.Run("adds property to schema", func(t *testing.T) {
		schema := NewConfigSchema("Test", "")
		prop := PropertySchema{Type: "string", Title: "Name"}

		result := schema.AddProperty("name", prop)

		assert.Same(t, &schema, result) // Returns same schema for chaining
		assert.Contains(t, schema.Properties, "name")
		assert.Equal(t, prop, schema.Properties["name"])
	})

	t.Run("handles nil properties map", func(t *testing.T) {
		schema := ConfigSchema{}

		schema.AddProperty("key", PropertySchema{Type: "string"})

		assert.NotNil(t, schema.Properties)
		assert.Contains(t, schema.Properties, "key")
	})
}

func TestConfigSchema_AddRequired(t *testing.T) {
	t.Run("adds required field", func(t *testing.T) {
		schema := NewConfigSchema("Test", "")

		result := schema.AddRequired("name").AddRequired("email")

		assert.Same(t, &schema, result)
		assert.Contains(t, schema.Required, "name")
		assert.Contains(t, schema.Required, "email")
	})
}

func TestConfigSchema_Validate(t *testing.T) {
	schema := NewConfigSchema("Test", "")
	schema.AddProperty("name", PropertySchema{Type: "string"})
	schema.AddProperty("age", PropertySchema{Type: "integer", Minimum: FloatPtr(0)})
	schema.AddRequired("name")

	t.Run("valid config passes", func(t *testing.T) {
		config := map[string]any{"name": "Alice", "age": 30}

		err := schema.Validate(config)

		assert.NoError(t, err)
	})

	t.Run("fails on missing required field", func(t *testing.T) {
		config := map[string]any{"age": 30}

		err := schema.Validate(config)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("allows unknown properties", func(t *testing.T) {
		config := map[string]any{"name": "Alice", "unknown": "value"}

		err := schema.Validate(config)

		assert.NoError(t, err)
	})

	t.Run("validates property types", func(t *testing.T) {
		config := map[string]any{"name": 123} // Wrong type

		err := schema.Validate(config)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "string")
	})
}

func TestPropertySchema_Validate(t *testing.T) {
	t.Run("validates string type", func(t *testing.T) {
		prop := PropertySchema{Type: "string"}

		assert.NoError(t, prop.Validate("field", "valid"))
		assert.Error(t, prop.Validate("field", 123))
	})

	t.Run("validates string minLength", func(t *testing.T) {
		prop := PropertySchema{Type: "string", MinLength: IntPtr(3)}

		assert.NoError(t, prop.Validate("field", "abc"))
		assert.Error(t, prop.Validate("field", "ab"))
	})

	t.Run("validates string maxLength", func(t *testing.T) {
		prop := PropertySchema{Type: "string", MaxLength: IntPtr(5)}

		assert.NoError(t, prop.Validate("field", "abcde"))
		assert.Error(t, prop.Validate("field", "abcdef"))
	})

	t.Run("validates number type", func(t *testing.T) {
		prop := PropertySchema{Type: "number"}

		assert.NoError(t, prop.Validate("field", 3.14))
		assert.NoError(t, prop.Validate("field", float32(2.5)))
		assert.NoError(t, prop.Validate("field", 42))
		assert.NoError(t, prop.Validate("field", int64(100)))
		assert.Error(t, prop.Validate("field", "string"))
	})

	t.Run("validates number minimum", func(t *testing.T) {
		prop := PropertySchema{Type: "number", Minimum: FloatPtr(0)}

		assert.NoError(t, prop.Validate("field", 0.0))
		assert.NoError(t, prop.Validate("field", 10.0))
		assert.Error(t, prop.Validate("field", -1.0))
	})

	t.Run("validates number maximum", func(t *testing.T) {
		prop := PropertySchema{Type: "number", Maximum: FloatPtr(100)}

		assert.NoError(t, prop.Validate("field", 100.0))
		assert.NoError(t, prop.Validate("field", 50.0))
		assert.Error(t, prop.Validate("field", 101.0))
	})

	t.Run("validates integer type", func(t *testing.T) {
		prop := PropertySchema{Type: "integer"}

		assert.NoError(t, prop.Validate("field", 42))
		assert.Error(t, prop.Validate("field", "string"))
	})

	t.Run("validates boolean type", func(t *testing.T) {
		prop := PropertySchema{Type: "boolean"}

		assert.NoError(t, prop.Validate("field", true))
		assert.NoError(t, prop.Validate("field", false))
		assert.Error(t, prop.Validate("field", "true"))
	})

	t.Run("validates array type", func(t *testing.T) {
		prop := PropertySchema{Type: "array"}

		assert.NoError(t, prop.Validate("field", []any{"a", "b"}))
		assert.Error(t, prop.Validate("field", "not-array"))
	})

	t.Run("validates enum constraint", func(t *testing.T) {
		prop := PropertySchema{Type: "string", Enum: []any{"a", "b", "c"}}

		assert.NoError(t, prop.Validate("field", "a"))
		assert.NoError(t, prop.Validate("field", "c"))
		assert.Error(t, prop.Validate("field", "d"))
	})

	t.Run("allows nil value", func(t *testing.T) {
		prop := PropertySchema{Type: "string"}

		assert.NoError(t, prop.Validate("field", nil))
	})
}

func TestFloatPtr(t *testing.T) {
	t.Run("returns pointer to float", func(t *testing.T) {
		ptr := FloatPtr(3.14)

		require.NotNil(t, ptr)
		assert.Equal(t, 3.14, *ptr)
	})
}

func TestIntPtr(t *testing.T) {
	t.Run("returns pointer to int", func(t *testing.T) {
		ptr := IntPtr(42)

		require.NotNil(t, ptr)
		assert.Equal(t, 42, *ptr)
	})
}
