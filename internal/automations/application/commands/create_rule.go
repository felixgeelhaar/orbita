package commands

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
)

// CreateRuleCommand contains the data needed to create an automation rule.
type CreateRuleCommand struct {
	UserID      uuid.UUID
	Name        string
	Description string

	TriggerType   domain.TriggerType
	TriggerConfig map[string]any

	Conditions        []types.RuleCondition
	ConditionOperator domain.ConditionOperator

	Actions []types.RuleAction

	CooldownSeconds      int
	MaxExecutionsPerHour *int
	Priority             int
	Tags                 []string
}

// Validate validates the command.
func (c CreateRuleCommand) Validate() error {
	if c.UserID == uuid.Nil {
		return errors.New("user_id is required")
	}
	if c.Name == "" {
		return errors.New("name is required")
	}
	if c.TriggerType == "" {
		return errors.New("trigger_type is required")
	}
	if len(c.Actions) == 0 {
		return errors.New("at least one action is required")
	}
	return nil
}

// CreateRuleHandler handles the CreateRuleCommand.
type CreateRuleHandler struct {
	ruleRepo domain.RuleRepository
}

// NewCreateRuleHandler creates a new CreateRuleHandler.
func NewCreateRuleHandler(ruleRepo domain.RuleRepository) *CreateRuleHandler {
	return &CreateRuleHandler{ruleRepo: ruleRepo}
}

// Handle executes the CreateRuleCommand.
func (h *CreateRuleHandler) Handle(ctx context.Context, cmd CreateRuleCommand) (*domain.AutomationRule, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	rule, err := domain.NewAutomationRule(
		cmd.UserID,
		cmd.Name,
		cmd.TriggerType,
		cmd.TriggerConfig,
		cmd.Actions,
	)
	if err != nil {
		return nil, err
	}

	if cmd.Description != "" {
		rule.SetDescription(cmd.Description)
	}
	if cmd.Priority != 0 {
		rule.SetPriority(cmd.Priority)
	}
	if len(cmd.Conditions) > 0 {
		operator := cmd.ConditionOperator
		if operator == "" {
			operator = domain.ConditionOperatorAND
		}
		rule.SetConditions(cmd.Conditions, operator)
	}
	if cmd.CooldownSeconds > 0 {
		rule.SetCooldown(cmd.CooldownSeconds)
	}
	if cmd.MaxExecutionsPerHour != nil {
		rule.SetMaxExecutionsPerHour(cmd.MaxExecutionsPerHour)
	}
	for _, tag := range cmd.Tags {
		rule.AddTag(tag)
	}

	if err := h.ruleRepo.Create(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}
