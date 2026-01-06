package mcp

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/google/uuid"
)

type authExchangeInput struct {
	Code string `json:"code" jsonschema:"required"`
}

func registerAuthTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App
	service := deps.AuthService

	srv.Tool("auth.url").
		Description("Generate OAuth2 authorization URL").
		Handler(func(ctx context.Context, input struct{}) (map[string]any, error) {
			if service == nil {
				return nil, errors.New("auth service not configured")
			}
			state := uuid.New().String()
			url := service.AuthURL(state)
			return map[string]any{
				"url":   url,
				"state": state,
			}, nil
		})

	srv.Tool("auth.exchange").
		Description("Exchange OAuth2 code for tokens and store them").
		Handler(func(ctx context.Context, input authExchangeInput) (map[string]any, error) {
			if service == nil {
				return nil, errors.New("auth service not configured")
			}
			if app == nil || app.CurrentUserID == uuid.Nil {
				return nil, errors.New("current user not configured")
			}
			if input.Code == "" {
				return nil, errors.New("code is required")
			}
			if _, err := service.ExchangeAndStore(ctx, app.CurrentUserID, input.Code); err != nil {
				return nil, err
			}
			return map[string]any{"stored": true}, nil
		})

	return nil
}
