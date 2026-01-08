package enginesdk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/felixgeelhaar/orbita/internal/engine/grpc"
	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/hashicorp/go-plugin"
)

// Serve starts the plugin server for an engine.
// This should be called from the main function of a plugin binary.
//
// Example:
//
//	func main() {
//		enginesdk.Serve(&MyEngine{})
//	}
func Serve(engine sdk.Engine) {
	// Create the plugin map based on engine type
	pluginMap := grpc.PluginMapForEngine(engine.Type())
	if pluginMap == nil {
		panic("unsupported engine type")
	}

	// Serve the plugin
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: grpc.HandshakeConfig,
		Plugins:         pluginMap,
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}

// ServeScheduler starts the plugin server for a scheduler engine.
func ServeScheduler(engine SchedulerEngine) {
	Serve(engine)
}

// ServePriority starts the plugin server for a priority engine.
func ServePriority(engine PriorityEngine) {
	Serve(engine)
}

// ServeClassifier starts the plugin server for a classifier engine.
func ServeClassifier(engine ClassifierEngine) {
	Serve(engine)
}

// ServeAutomation starts the plugin server for an automation engine.
func ServeAutomation(engine AutomationEngine) {
	Serve(engine)
}

// PluginConfig holds plugin configuration loaded from the manifest.
type PluginConfig struct {
	// ManifestPath is the path to the engine.json manifest.
	ManifestPath string

	// Config contains user configuration values.
	Config map[string]any
}

// LoadConfig loads plugin configuration from environment and manifest.
// The ORBITA_ENGINE_CONFIG environment variable contains JSON-encoded config.
func LoadConfig() (*PluginConfig, error) {
	config := &PluginConfig{
		Config: make(map[string]any),
	}

	// Load config from environment variable
	configJSON := os.Getenv("ORBITA_ENGINE_CONFIG")
	if configJSON != "" {
		if err := json.Unmarshal([]byte(configJSON), &config.Config); err != nil {
			return nil, fmt.Errorf("failed to parse ORBITA_ENGINE_CONFIG: %w", err)
		}
	}

	// Get manifest path
	config.ManifestPath = os.Getenv("ORBITA_ENGINE_MANIFEST")

	return config, nil
}

// BaseEngine provides a default implementation of common Engine methods.
// Plugin developers can embed this struct to reduce boilerplate.
type BaseEngine struct {
	metadata sdk.EngineMetadata
	config   sdk.EngineConfig
}

// NewBaseEngine creates a new BaseEngine with the given metadata.
func NewBaseEngine(metadata sdk.EngineMetadata) *BaseEngine {
	return &BaseEngine{
		metadata: metadata,
	}
}

// Metadata returns the engine metadata.
func (e *BaseEngine) Metadata() sdk.EngineMetadata {
	return e.metadata
}

// ConfigSchema returns an empty configuration schema.
// Override this method to provide a custom schema.
func (e *BaseEngine) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{
		Schema:     "https://json-schema.org/draft/2020-12/schema",
		Properties: make(map[string]sdk.PropertySchema),
		Required:   []string{},
	}
}

// Initialize stores the configuration.
func (e *BaseEngine) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	e.config = config
	return nil
}

// HealthCheck returns a healthy status.
func (e *BaseEngine) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{
		Healthy: true,
		Message: "engine is healthy",
	}
}

// Shutdown is a no-op by default.
func (e *BaseEngine) Shutdown(ctx context.Context) error {
	return nil
}

// Config returns the engine configuration.
func (e *BaseEngine) Config() sdk.EngineConfig {
	return e.config
}

// GetString retrieves a string configuration value with a default.
func (e *BaseEngine) GetString(key, defaultVal string) string {
	if e.config.Has(key) {
		return e.config.GetString(key)
	}
	return defaultVal
}

// GetInt retrieves an integer configuration value with a default.
func (e *BaseEngine) GetInt(key string, defaultVal int) int {
	if e.config.Has(key) {
		return e.config.GetInt(key)
	}
	return defaultVal
}

// GetFloat retrieves a float configuration value with a default.
func (e *BaseEngine) GetFloat(key string, defaultVal float64) float64 {
	if e.config.Has(key) {
		return e.config.GetFloat(key)
	}
	return defaultVal
}

// GetBool retrieves a boolean configuration value with a default.
func (e *BaseEngine) GetBool(key string, defaultVal bool) bool {
	if e.config.Has(key) {
		return e.config.GetBool(key)
	}
	return defaultVal
}

// PropertyBuilder helps construct PropertySchema instances.
type PropertyBuilder struct {
	prop sdk.PropertySchema
}

// NewProperty creates a new property builder.
func NewProperty(propType, title, description string) *PropertyBuilder {
	return &PropertyBuilder{
		prop: sdk.PropertySchema{
			Type:        propType,
			Title:       title,
			Description: description,
		},
	}
}

// Default sets the default value.
func (b *PropertyBuilder) Default(value any) *PropertyBuilder {
	b.prop.Default = value
	return b
}

// Min sets the minimum value for numeric properties.
func (b *PropertyBuilder) Min(value float64) *PropertyBuilder {
	b.prop.Minimum = &value
	return b
}

// Max sets the maximum value for numeric properties.
func (b *PropertyBuilder) Max(value float64) *PropertyBuilder {
	b.prop.Maximum = &value
	return b
}

// Enum sets the allowed enum values.
func (b *PropertyBuilder) Enum(values ...any) *PropertyBuilder {
	b.prop.Enum = values
	return b
}

// Widget sets the UI widget hint.
func (b *PropertyBuilder) Widget(widget string) *PropertyBuilder {
	b.prop.UIHints.Widget = widget
	return b
}

// Group sets the UI group.
func (b *PropertyBuilder) Group(group string) *PropertyBuilder {
	b.prop.UIHints.Group = group
	return b
}

// Order sets the display order.
func (b *PropertyBuilder) Order(order int) *PropertyBuilder {
	b.prop.UIHints.Order = order
	return b
}

// HelpText sets the help text.
func (b *PropertyBuilder) HelpText(text string) *PropertyBuilder {
	b.prop.UIHints.HelpText = text
	return b
}

// Build returns the constructed PropertySchema.
func (b *PropertyBuilder) Build() sdk.PropertySchema {
	return b.prop
}

// ConfigSchemaBuilder helps construct ConfigSchema instances.
type ConfigSchemaBuilder struct {
	schema sdk.ConfigSchema
}

// NewConfigSchema creates a new configuration schema builder.
func NewConfigSchema() *ConfigSchemaBuilder {
	return &ConfigSchemaBuilder{
		schema: sdk.ConfigSchema{
			Schema:     "https://json-schema.org/draft/2020-12/schema",
			Properties: make(map[string]sdk.PropertySchema),
			Required:   []string{},
		},
	}
}

// AddProperty adds a property to the schema.
func (b *ConfigSchemaBuilder) AddProperty(name string, prop sdk.PropertySchema) *ConfigSchemaBuilder {
	b.schema.Properties[name] = prop
	return b
}

// Required marks properties as required.
func (b *ConfigSchemaBuilder) Required(names ...string) *ConfigSchemaBuilder {
	b.schema.Required = append(b.schema.Required, names...)
	return b
}

// Build returns the constructed ConfigSchema.
func (b *ConfigSchemaBuilder) Build() sdk.ConfigSchema {
	return b.schema
}

// MetadataBuilder helps construct EngineMetadata instances.
type MetadataBuilder struct {
	metadata sdk.EngineMetadata
}

// NewMetadata creates a new metadata builder.
func NewMetadata(id, name, version string) *MetadataBuilder {
	return &MetadataBuilder{
		metadata: sdk.EngineMetadata{
			ID:      id,
			Name:    name,
			Version: version,
		},
	}
}

// Author sets the author.
func (b *MetadataBuilder) Author(author string) *MetadataBuilder {
	b.metadata.Author = author
	return b
}

// Description sets the description.
func (b *MetadataBuilder) Description(desc string) *MetadataBuilder {
	b.metadata.Description = desc
	return b
}

// License sets the license.
func (b *MetadataBuilder) License(license string) *MetadataBuilder {
	b.metadata.License = license
	return b
}

// Homepage sets the homepage URL.
func (b *MetadataBuilder) Homepage(url string) *MetadataBuilder {
	b.metadata.Homepage = url
	return b
}

// Tags sets the tags.
func (b *MetadataBuilder) Tags(tags ...string) *MetadataBuilder {
	b.metadata.Tags = tags
	return b
}

// MinAPIVersion sets the minimum API version.
func (b *MetadataBuilder) MinAPIVersion(version string) *MetadataBuilder {
	b.metadata.MinAPIVersion = version
	return b
}

// Capabilities sets the capabilities.
func (b *MetadataBuilder) Capabilities(caps ...string) *MetadataBuilder {
	b.metadata.Capabilities = caps
	return b
}

// Build returns the constructed EngineMetadata.
func (b *MetadataBuilder) Build() sdk.EngineMetadata {
	return b.metadata
}
