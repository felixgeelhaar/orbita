package automation

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func resetFlags() {
	// List flags
	listEnabled = ""
	listTriggerType = ""
	listTags = nil
	listLimit = 50

	// Create flags
	createTriggerType = "event"
	createTriggerConfig = ""
	createConditions = ""
	createActions = ""
	createDescription = ""
	createPriority = 0
	createCooldown = 0
	createMaxPerHour = 0
	createTags = nil

	// Get flags
	getJSON = false

	// Delete flags
	deleteForce = false

	// Executions flags
	execRuleID = ""
	execStatus = ""
	execLimit = 20
}

// Test commands when app is nil or AutomationService is nil
// These commands return nil (no error) and print a message to stdout
func TestListCmd_NoApp(t *testing.T) {
	resetFlags()
	cli.SetApp(nil)

	listCmd.SetContext(context.Background())

	// Returns nil, just prints message to stdout
	err := listCmd.RunE(listCmd, []string{})
	assert.NoError(t, err)
}

func TestListCmd_NoAutomationService(t *testing.T) {
	resetFlags()
	app := &cli.App{
		CurrentUserID:     uuid.New(),
		AutomationService: nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	listCmd.SetContext(context.Background())

	err := listCmd.RunE(listCmd, []string{})
	assert.NoError(t, err)
}

func TestCreateCmd_NoApp(t *testing.T) {
	resetFlags()
	cli.SetApp(nil)

	createCmd.SetContext(context.Background())

	err := createCmd.RunE(createCmd, []string{"Test Rule"})
	assert.NoError(t, err)
}

func TestCreateCmd_NoAutomationService(t *testing.T) {
	resetFlags()
	app := &cli.App{
		CurrentUserID:     uuid.New(),
		AutomationService: nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	createCmd.SetContext(context.Background())

	err := createCmd.RunE(createCmd, []string{"Test Rule"})
	assert.NoError(t, err)
}

func TestEnableCmd_NoApp(t *testing.T) {
	resetFlags()
	cli.SetApp(nil)

	enableCmd.SetContext(context.Background())

	err := enableCmd.RunE(enableCmd, []string{uuid.NewString()})
	assert.NoError(t, err)
}

func TestEnableCmd_NoAutomationService(t *testing.T) {
	resetFlags()
	app := &cli.App{
		CurrentUserID:     uuid.New(),
		AutomationService: nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	enableCmd.SetContext(context.Background())

	err := enableCmd.RunE(enableCmd, []string{uuid.NewString()})
	assert.NoError(t, err)
}

func TestDisableCmd_NoApp(t *testing.T) {
	resetFlags()
	cli.SetApp(nil)

	disableCmd.SetContext(context.Background())

	err := disableCmd.RunE(disableCmd, []string{uuid.NewString()})
	assert.NoError(t, err)
}

func TestDisableCmd_NoAutomationService(t *testing.T) {
	resetFlags()
	app := &cli.App{
		CurrentUserID:     uuid.New(),
		AutomationService: nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	disableCmd.SetContext(context.Background())

	err := disableCmd.RunE(disableCmd, []string{uuid.NewString()})
	assert.NoError(t, err)
}

func TestGetCmd_NoApp(t *testing.T) {
	resetFlags()
	cli.SetApp(nil)

	getCmd.SetContext(context.Background())

	err := getCmd.RunE(getCmd, []string{uuid.NewString()})
	assert.NoError(t, err)
}

func TestGetCmd_NoAutomationService(t *testing.T) {
	resetFlags()
	app := &cli.App{
		CurrentUserID:     uuid.New(),
		AutomationService: nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	getCmd.SetContext(context.Background())

	err := getCmd.RunE(getCmd, []string{uuid.NewString()})
	assert.NoError(t, err)
}

func TestDeleteCmd_NoApp(t *testing.T) {
	resetFlags()
	cli.SetApp(nil)

	deleteCmd.SetContext(context.Background())

	err := deleteCmd.RunE(deleteCmd, []string{uuid.NewString()})
	assert.NoError(t, err)
}

func TestDeleteCmd_NoAutomationService(t *testing.T) {
	resetFlags()
	app := &cli.App{
		CurrentUserID:     uuid.New(),
		AutomationService: nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	deleteCmd.SetContext(context.Background())
	deleteForce = true

	err := deleteCmd.RunE(deleteCmd, []string{uuid.NewString()})
	assert.NoError(t, err)
}

func TestExecutionsCmd_NoApp(t *testing.T) {
	resetFlags()
	cli.SetApp(nil)

	executionsCmd.SetContext(context.Background())

	err := executionsCmd.RunE(executionsCmd, []string{})
	assert.NoError(t, err)
}

func TestExecutionsCmd_NoAutomationService(t *testing.T) {
	resetFlags()
	app := &cli.App{
		CurrentUserID:     uuid.New(),
		AutomationService: nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	executionsCmd.SetContext(context.Background())

	err := executionsCmd.RunE(executionsCmd, []string{})
	assert.NoError(t, err)
}

// Test helper functions
func TestStatusText(t *testing.T) {
	tests := []struct {
		enabled  bool
		expected string
	}{
		{true, "enabled"},
		{false, "disabled"},
	}

	for _, tc := range tests {
		result := statusText(tc.enabled)
		assert.Equal(t, tc.expected, result)
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status   domain.ExecutionStatus
		expected string
	}{
		{domain.ExecutionStatusSuccess, "✓"},
		{domain.ExecutionStatusFailed, "✗"},
		{domain.ExecutionStatusSkipped, "○"},
		{domain.ExecutionStatusPartial, "◐"},
		{domain.ExecutionStatusPending, "◷"},
		{domain.ExecutionStatus("unknown"), "?"},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			result := getStatusIcon(tc.status)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Test command configuration
func TestCmdConfiguration(t *testing.T) {
	// Verify command structure
	assert.Equal(t, "automation", Cmd.Use)
	assert.Equal(t, []string{"auto", "rule"}, Cmd.Aliases)

	// Verify subcommands are registered
	subCmds := Cmd.Commands()
	cmdNames := make([]string, len(subCmds))
	for i, cmd := range subCmds {
		cmdNames[i] = cmd.Use
	}

	assert.Contains(t, cmdNames, "create [name]")
	assert.Contains(t, cmdNames, "list")
	assert.Contains(t, cmdNames, "get [rule-id]")
	assert.Contains(t, cmdNames, "enable [rule-id]")
	assert.Contains(t, cmdNames, "disable [rule-id]")
	assert.Contains(t, cmdNames, "delete [rule-id]")
	assert.Contains(t, cmdNames, "executions")
}

func TestListCmdAliases(t *testing.T) {
	assert.Equal(t, "list", listCmd.Use)
	assert.Equal(t, []string{"ls"}, listCmd.Aliases)
}

func TestGetCmdAliases(t *testing.T) {
	assert.Equal(t, "get [rule-id]", getCmd.Use)
	assert.Equal(t, []string{"show", "info"}, getCmd.Aliases)
}

func TestDeleteCmdAliases(t *testing.T) {
	assert.Equal(t, "delete [rule-id]", deleteCmd.Use)
	assert.Equal(t, []string{"rm", "remove"}, deleteCmd.Aliases)
}

func TestExecutionsCmdAliases(t *testing.T) {
	assert.Equal(t, "executions", executionsCmd.Use)
	assert.Equal(t, []string{"exec", "history"}, executionsCmd.Aliases)
}

func TestListCmdFlags(t *testing.T) {
	resetFlags()

	// Verify flags exist
	assert.NotNil(t, listCmd.Flags().Lookup("enabled"))
	assert.NotNil(t, listCmd.Flags().Lookup("trigger"))
	assert.NotNil(t, listCmd.Flags().Lookup("tags"))
	assert.NotNil(t, listCmd.Flags().Lookup("limit"))
}

func TestCreateCmdFlags(t *testing.T) {
	resetFlags()

	// Verify flags exist
	assert.NotNil(t, createCmd.Flags().Lookup("trigger-type"))
	assert.NotNil(t, createCmd.Flags().Lookup("trigger-config"))
	assert.NotNil(t, createCmd.Flags().Lookup("conditions"))
	assert.NotNil(t, createCmd.Flags().Lookup("actions"))
	assert.NotNil(t, createCmd.Flags().Lookup("description"))
	assert.NotNil(t, createCmd.Flags().Lookup("priority"))
	assert.NotNil(t, createCmd.Flags().Lookup("cooldown"))
	assert.NotNil(t, createCmd.Flags().Lookup("max-per-hour"))
	assert.NotNil(t, createCmd.Flags().Lookup("tags"))
}

func TestGetCmdFlags(t *testing.T) {
	resetFlags()

	// Verify flags exist
	assert.NotNil(t, getCmd.Flags().Lookup("json"))
}

func TestDeleteCmdFlags(t *testing.T) {
	resetFlags()

	// Verify flags exist
	assert.NotNil(t, deleteCmd.Flags().Lookup("force"))
}

func TestExecutionsCmdFlags(t *testing.T) {
	resetFlags()

	// Verify flags exist
	assert.NotNil(t, executionsCmd.Flags().Lookup("rule"))
	assert.NotNil(t, executionsCmd.Flags().Lookup("status"))
	assert.NotNil(t, executionsCmd.Flags().Lookup("limit"))
}
