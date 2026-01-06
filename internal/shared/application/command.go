package application

import "context"

// Command represents a command that modifies system state.
type Command interface {
	CommandName() string
}

// CommandHandler handles a specific command type.
type CommandHandler[C Command] interface {
	Handle(ctx context.Context, cmd C) error
}

// CommandResult represents the result of a command execution.
type CommandResult struct {
	Success bool
	Error   error
	Data    any
}

// NewSuccessResult creates a successful command result.
func NewSuccessResult(data any) CommandResult {
	return CommandResult{Success: true, Data: data}
}

// NewErrorResult creates a failed command result.
func NewErrorResult(err error) CommandResult {
	return CommandResult{Success: false, Error: err}
}
