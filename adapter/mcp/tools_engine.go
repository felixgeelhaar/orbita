package mcp

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
)

// EngineDTO represents an engine in MCP responses.
type EngineDTO struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Version     string   `json:"version"`
	Status      string   `json:"status"`
	Builtin     bool     `json:"builtin"`
	Author      string   `json:"author,omitempty"`
	Description string   `json:"description,omitempty"`
	License     string   `json:"license,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type engineListInput struct {
	Type string `json:"type,omitempty"`
}

type engineIDInput struct {
	EngineID string `json:"engine_id" jsonschema:"required"`
}

type engineHealthInput struct {
	EngineID string `json:"engine_id" jsonschema:"required"`
}

func registerEngineTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("engine.list").
		Description("List all registered engines with their status").
		Handler(func(ctx context.Context, input engineListInput) ([]EngineDTO, error) {
			if app == nil || app.EngineRegistry == nil {
				return nil, errors.New("engine registry not available")
			}

			// Get all entries from registry
			allEntries := app.EngineRegistry.List()
			result := make([]EngineDTO, 0, len(allEntries))

			for _, entry := range allEntries {
				// Filter by type if specified
				if input.Type != "" && entry.Manifest != nil && entry.Manifest.Type != input.Type {
					continue
				}

				dto := EngineDTO{
					Status:  string(entry.Status),
					Builtin: entry.Builtin,
				}

				// Get metadata from engine or manifest
				if entry.Engine != nil {
					metadata := entry.Engine.Metadata()
					dto.ID = metadata.ID
					dto.Name = metadata.Name
					dto.Type = entry.Engine.Type().String()
					dto.Version = metadata.Version
					dto.Author = metadata.Author
					dto.Description = metadata.Description
					dto.License = metadata.License
					dto.Homepage = metadata.Homepage
					dto.Tags = metadata.Tags
				} else if entry.Manifest != nil {
					dto.ID = entry.Manifest.ID
					dto.Name = entry.Manifest.Name
					dto.Type = entry.Manifest.Type
					dto.Version = entry.Manifest.Version
					dto.Author = entry.Manifest.Author
					dto.Description = entry.Manifest.Description
					dto.License = entry.Manifest.License
					dto.Homepage = entry.Manifest.Homepage
				}

				result = append(result, dto)
			}

			return result, nil
		})

	srv.Tool("engine.info").
		Description("Get detailed information about a specific engine").
		Handler(func(ctx context.Context, input engineIDInput) (*EngineDTO, error) {
			if app == nil || app.EngineRegistry == nil {
				return nil, errors.New("engine registry not available")
			}

			if input.EngineID == "" {
				return nil, errors.New("engine_id is required")
			}

			metadata, err := app.EngineRegistry.GetMetadata(input.EngineID)
			if err != nil {
				return nil, err
			}

			status, _ := app.EngineRegistry.Status(input.EngineID)

			// Check if builtin by listing and finding
			entries := app.EngineRegistry.List()
			var builtin bool
			var engineType string
			for _, e := range entries {
				if e.Manifest != nil && e.Manifest.ID == input.EngineID {
					builtin = e.Builtin
					engineType = e.Manifest.Type
					break
				}
			}

			return &EngineDTO{
				ID:          metadata.ID,
				Name:        metadata.Name,
				Type:        engineType,
				Version:     metadata.Version,
				Status:      string(status),
				Builtin:     builtin,
				Author:      metadata.Author,
				Description: metadata.Description,
				License:     metadata.License,
				Homepage:    metadata.Homepage,
				Tags:        metadata.Tags,
			}, nil
		})

	srv.Tool("engine.health").
		Description("Check the health of a specific engine").
		Handler(func(ctx context.Context, input engineHealthInput) (map[string]any, error) {
			if app == nil || app.EngineRegistry == nil {
				return nil, errors.New("engine registry not available")
			}

			if input.EngineID == "" {
				return nil, errors.New("engine_id is required")
			}

			engine, err := app.EngineRegistry.Get(ctx, input.EngineID)
			if err != nil {
				return nil, err
			}

			health := engine.HealthCheck(ctx)

			return map[string]any{
				"engine_id":  input.EngineID,
				"healthy":    health.Healthy,
				"message":    health.Message,
				"details":    health.Details,
				"checked_at": health.CheckedAt,
			}, nil
		})

	srv.Tool("engine.types").
		Description("List available engine types").
		Handler(func(ctx context.Context, input struct{}) ([]map[string]string, error) {
			return []map[string]string{
				{"type": sdk.EngineTypePriority.String(), "description": "Priority scoring engines for task ranking"},
				{"type": sdk.EngineTypeScheduler.String(), "description": "Scheduling engines for time-blocking"},
				{"type": sdk.EngineTypeClassifier.String(), "description": "Classification engines for inbox categorization"},
				{"type": sdk.EngineTypeAutomation.String(), "description": "Automation engines for rule evaluation"},
			}, nil
		})

	return nil
}
