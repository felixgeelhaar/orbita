package commands

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
)

// UpdateRuleCommand contains the data needed to update an automation rule.
type UpdateRuleCommand struct {
	RuleID uuid.UUID
	UserID uuid.UUID // For authorization

	Name        *string
	Description *string
	Enabled     *bool
	Priority    *int

	TriggerType   *domain.TriggerType
	TriggerConfig map[string]any

	Conditions        []types.RuleCondition
	ConditionOperator *domain.ConditionOperator

	Actions []types.RuleAction

	CooldownSeconds      *int
	MaxExecutionsPerHour *int
	Tags                 []string
}

// Validate validates the command.
func (c UpdateRuleCommand) Validate() error {
	if c.RuleID == uuid.Nil {
		return errors.New("rule_id is required")
	}
	if c.UserID == uuid.Nil {
		return errors.New("user_id is required")
	}
	return nil
}

// UpdateRuleHandler handles the UpdateRuleCommand.
type UpdateRuleHandler struct {
	ruleRepo domain.RuleRepository
}

// NewUpdateRuleHandler creates a new UpdateRuleHandler.
func NewUpdateRuleHandler(ruleRepo domain.RuleRepository) *UpdateRuleHandler {
	return &UpdateRuleHandler{ruleRepo: ruleRepo}
}

// Handle executes the UpdateRuleCommand.
func (h *UpdateRuleHandler) Handle(ctx context.Context, cmd UpdateRuleCommand) (*domain.AutomationRule, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	rule, err := h.ruleRepo.GetByID(ctx, cmd.RuleID)
	if err != nil {
		return nil, err
	}

	// Authorization check
	if rule.UserID != cmd.UserID {
		return nil, domain.ErrRuleNotFound
	}

	// Apply updates
	if cmd.Name != nil && *cmd.Name != "" {
		rule.Name = *cmd.Name
	}
	if cmd.Description != nil {
		rule.SetDescription(*cmd.Description)
	}
	if cmd.Enabled != nil {
		if *cmd.Enabled {
			rule.Enable()
		} else {
			rule.Disable()
		}
	}
	if cmd.Priority != nil {
		rule.SetPriority(*cmd.Priority)
	}
	if cmd.TriggerType != nil {
		rule.TriggerType = *cmd.TriggerType
	}
	if cmd.TriggerConfig != nil {
		rule.TriggerConfig = cmd.TriggerConfig
	}
	if cmd.Conditions != nil {
		operator := domain.ConditionOperatorAND
		if cmd.ConditionOperator != nil {
			operator = *cmd.ConditionOperator
		}
		rule.SetConditions(cmd.Conditions, operator)
	}
	if cmd.Actions != nil && len(cmd.Actions) > 0 {
		rule.Actions = cmd.Actions
	}
	if cmd.CooldownSeconds != nil {
		rule.SetCooldown(*cmd.CooldownSeconds)
	}
	if cmd.MaxExecutionsPerHour != nil {
		rule.SetMaxExecutionsPerHour(cmd.MaxExecutionsPerHour)
	}
	if cmd.Tags != nil {
		rule.Tags = []string{}
		for _, tag := range cmd.Tags {
			rule.AddTag(tag)
		}
	}

	if err := h.ruleRepo.Update(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}
