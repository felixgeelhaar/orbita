package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/felixgeelhaar/mcp-go"
)

type billingGrantInput struct {
	Module string `json:"module" jsonschema:"required"`
	Active bool   `json:"active,omitempty"`
	Source string `json:"source,omitempty"`
}

type billingWebhookInput struct {
	EventPath string `json:"event_path,omitempty"`
	EventJSON string `json:"event_json,omitempty"`
}

func registerBillingTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("billing.status").
		Description("Get subscription status").
		Handler(func(ctx context.Context, input struct{}) (any, error) {
			if app == nil || app.BillingService == nil {
				return nil, errors.New("billing status requires database connection")
			}
			return app.BillingService.GetSubscription(ctx, app.CurrentUserID)
		})

	srv.Tool("billing.entitlements").
		Description("List entitlements").
		Handler(func(ctx context.Context, input struct{}) (any, error) {
			if app == nil || app.BillingService == nil {
				return nil, errors.New("entitlements require database connection")
			}
			return app.BillingService.ListEntitlements(ctx, app.CurrentUserID)
		})

	srv.Tool("billing.grant").
		Description("Grant or revoke an entitlement").
		Handler(func(ctx context.Context, input billingGrantInput) (map[string]any, error) {
			if app == nil || app.BillingService == nil {
				return nil, errors.New("entitlement updates require database connection")
			}
			if input.Module == "" {
				return nil, errors.New("module is required")
			}
			if input.Source == "" {
				input.Source = "manual"
			}

			if err := app.BillingService.SetEntitlement(ctx, app.CurrentUserID, input.Module, input.Active, input.Source); err != nil {
				return nil, err
			}
			return map[string]any{"module": input.Module, "active": input.Active}, nil
		})

	srv.Tool("billing.webhook").
		Description("Handle a billing webhook payload").
		Handler(func(ctx context.Context, input billingWebhookInput) (map[string]any, error) {
			payload, err := loadWebhookPayload(input.EventPath, input.EventJSON)
			if err != nil {
				return nil, err
			}

			var envelope map[string]any
			if err := json.Unmarshal(payload, &envelope); err != nil {
				return nil, err
			}

			eventType, _ := envelope["type"].(string)
			if eventType == "" {
				eventType = "unknown"
			}

			return map[string]any{"event_type": eventType}, nil
		})

	return nil
}

func loadWebhookPayload(path string, payload string) ([]byte, error) {
	if payload != "" {
		return []byte(payload), nil
	}
	if path == "" {
		return nil, errors.New("event_path or event_json is required")
	}
	return os.ReadFile(path)
}
