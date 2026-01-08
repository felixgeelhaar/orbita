package sdk

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ConfigSchema defines the configuration structure using JSON Schema.
// This enables auto-generated UI for marketplace configuration.
type ConfigSchema struct {
	// Schema is the JSON Schema version (e.g., "https://json-schema.org/draft/2020-12/schema").
	Schema string `json:"$schema,omitempty"`

	// Type is the root type (always "object" for engine configs).
	Type string `json:"type"`

	// Title is a human-readable title for the configuration.
	Title string `json:"title"`

	// Description provides context about the configuration.
	Description string `json:"description,omitempty"`

	// Properties defines individual configuration fields.
	Properties map[string]PropertySchema `json:"properties"`

	// Required lists required property names.
	Required []string `json:"required,omitempty"`

	// Defaults provides default values for properties.
	Defaults map[string]any `json:"default,omitempty"`
}

// PropertySchema defines a single configuration property using JSON Schema.
type PropertySchema struct {
	// Type is the JSON Schema type: "string", "number", "integer", "boolean", "array", "object".
	Type string `json:"type"`

	// Title is a human-readable label for the property.
	Title string `json:"title"`

	// Description explains what the property controls.
	Description string `json:"description,omitempty"`

	// Default is the default value for the property.
	Default any `json:"default,omitempty"`

	// Enum restricts values to a specific set.
	Enum []any `json:"enum,omitempty"`

	// Minimum is the minimum value for numbers.
	Minimum *float64 `json:"minimum,omitempty"`

	// Maximum is the maximum value for numbers.
	Maximum *float64 `json:"maximum,omitempty"`

	// ExclusiveMinimum is the exclusive minimum for numbers.
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty"`

	// ExclusiveMaximum is the exclusive maximum for numbers.
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty"`

	// MinLength is the minimum string length.
	MinLength *int `json:"minLength,omitempty"`

	// MaxLength is the maximum string length.
	MaxLength *int `json:"maxLength,omitempty"`

	// Pattern is a regex pattern for string validation.
	Pattern string `json:"pattern,omitempty"`

	// Format is a semantic format hint (e.g., "email", "uri", "date-time").
	Format string `json:"format,omitempty"`

	// Items defines the schema for array items.
	Items *PropertySchema `json:"items,omitempty"`

	// UIHints provides rendering hints for marketplace UI.
	UIHints UIHints `json:"x-ui-hints,omitempty"`
}

// UIHints provides rendering hints for marketplace UI generation.
type UIHints struct {
	// Widget specifies the UI widget type.
	// Options: "text", "textarea", "slider", "toggle", "select", "multiselect", "color", "date", "time"
	Widget string `json:"widget,omitempty"`

	// Placeholder is placeholder text for input fields.
	Placeholder string `json:"placeholder,omitempty"`

	// HelpText provides additional guidance shown below the field.
	HelpText string `json:"helpText,omitempty"`

	// Group groups related fields together in the UI.
	Group string `json:"group,omitempty"`

	// Order controls display order (lower numbers first).
	Order int `json:"order,omitempty"`

	// Advanced hides the field in basic view.
	Advanced bool `json:"advanced,omitempty"`

	// Disabled makes the field read-only.
	Disabled bool `json:"disabled,omitempty"`

	// Visible controls conditional visibility.
	Visible *VisibilityCondition `json:"visible,omitempty"`
}

// VisibilityCondition defines when a field should be visible.
type VisibilityCondition struct {
	// Field is the field name to check.
	Field string `json:"field"`

	// Operator is the comparison operator: "eq", "ne", "gt", "lt", "in".
	Operator string `json:"operator"`

	// Value is the value to compare against.
	Value any `json:"value"`
}

// EngineConfig holds validated configuration values for an engine.
type EngineConfig struct {
	// Raw contains the raw configuration map.
	Raw map[string]any `json:"raw"`

	// UserID is the user this configuration applies to.
	UserID uuid.UUID `json:"user_id"`

	// EngineID identifies which engine this config is for.
	EngineID string `json:"engine_id"`
}

// NewEngineConfig creates a new engine configuration.
func NewEngineConfig(engineID string, userID uuid.UUID, raw map[string]any) EngineConfig {
	if raw == nil {
		raw = make(map[string]any)
	}
	return EngineConfig{
		Raw:      raw,
		UserID:   userID,
		EngineID: engineID,
	}
}

// Get retrieves a configuration value by key.
func (c EngineConfig) Get(key string) any {
	return c.Raw[key]
}

// GetString retrieves a string configuration value.
func (c EngineConfig) GetString(key string) string {
	if v, ok := c.Raw[key].(string); ok {
		return v
	}
	return ""
}

// GetInt retrieves an integer configuration value.
func (c EngineConfig) GetInt(key string) int {
	switch v := c.Raw[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
	}
	return 0
}

// GetFloat retrieves a float configuration value.
func (c EngineConfig) GetFloat(key string) float64 {
	switch v := c.Raw[key].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	}
	return 0
}

// GetBool retrieves a boolean configuration value.
func (c EngineConfig) GetBool(key string) bool {
	if v, ok := c.Raw[key].(bool); ok {
		return v
	}
	return false
}

// GetDuration retrieves a duration configuration value.
// The value should be a string parseable by time.ParseDuration.
func (c EngineConfig) GetDuration(key string) time.Duration {
	if v, ok := c.Raw[key].(string); ok {
		d, _ := time.ParseDuration(v)
		return d
	}
	return 0
}

// GetStringSlice retrieves a string slice configuration value.
func (c EngineConfig) GetStringSlice(key string) []string {
	if v, ok := c.Raw[key].([]any); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// Has checks if a configuration key exists.
func (c EngineConfig) Has(key string) bool {
	_, ok := c.Raw[key]
	return ok
}

// Merge merges another config into this one.
// Values from other take precedence.
func (c EngineConfig) Merge(other EngineConfig) EngineConfig {
	merged := make(map[string]any, len(c.Raw)+len(other.Raw))
	for k, v := range c.Raw {
		merged[k] = v
	}
	for k, v := range other.Raw {
		merged[k] = v
	}
	return EngineConfig{
		Raw:      merged,
		UserID:   c.UserID,
		EngineID: c.EngineID,
	}
}

// NewConfigSchema creates a new configuration schema with sensible defaults.
func NewConfigSchema(title, description string) ConfigSchema {
	return ConfigSchema{
		Schema:      "https://json-schema.org/draft/2020-12/schema",
		Type:        "object",
		Title:       title,
		Description: description,
		Properties:  make(map[string]PropertySchema),
	}
}

// AddProperty adds a property to the schema.
func (s *ConfigSchema) AddProperty(name string, prop PropertySchema) *ConfigSchema {
	if s.Properties == nil {
		s.Properties = make(map[string]PropertySchema)
	}
	s.Properties[name] = prop
	return s
}

// AddRequired marks a property as required.
func (s *ConfigSchema) AddRequired(name string) *ConfigSchema {
	s.Required = append(s.Required, name)
	return s
}

// Validate validates a configuration against this schema.
// Returns an error if validation fails.
func (s ConfigSchema) Validate(config map[string]any) error {
	// Check required fields
	for _, req := range s.Required {
		if _, ok := config[req]; !ok {
			return fmt.Errorf("required field %q is missing", req)
		}
	}

	// Validate each property
	for name, value := range config {
		prop, ok := s.Properties[name]
		if !ok {
			continue // Allow unknown properties
		}

		if err := prop.Validate(name, value); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates a value against this property schema.
func (p PropertySchema) Validate(name string, value any) error {
	if value == nil {
		return nil // Nil values are handled by required check
	}

	switch p.Type {
	case "string":
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("property %q must be a string", name)
		}
		if p.MinLength != nil && len(s) < *p.MinLength {
			return fmt.Errorf("property %q must be at least %d characters", name, *p.MinLength)
		}
		if p.MaxLength != nil && len(s) > *p.MaxLength {
			return fmt.Errorf("property %q must be at most %d characters", name, *p.MaxLength)
		}

	case "number", "integer":
		var f float64
		switch v := value.(type) {
		case float64:
			f = v
		case float32:
			f = float64(v)
		case int:
			f = float64(v)
		case int64:
			f = float64(v)
		default:
			return fmt.Errorf("property %q must be a number", name)
		}
		if p.Minimum != nil && f < *p.Minimum {
			return fmt.Errorf("property %q must be >= %v", name, *p.Minimum)
		}
		if p.Maximum != nil && f > *p.Maximum {
			return fmt.Errorf("property %q must be <= %v", name, *p.Maximum)
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("property %q must be a boolean", name)
		}

	case "array":
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("property %q must be an array", name)
		}
	}

	// Check enum constraint
	if len(p.Enum) > 0 {
		found := false
		for _, e := range p.Enum {
			if e == value {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("property %q must be one of %v", name, p.Enum)
		}
	}

	return nil
}

// FloatPtr returns a pointer to a float64 value.
// Helper for setting Minimum/Maximum in PropertySchema.
func FloatPtr(f float64) *float64 {
	return &f
}

// IntPtr returns a pointer to an int value.
// Helper for setting MinLength/MaxLength in PropertySchema.
func IntPtr(i int) *int {
	return &i
}
