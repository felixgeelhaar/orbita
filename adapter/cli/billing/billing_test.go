package billing

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetFlags() {
	grantModule = ""
	grantActive = true
	grantSource = "manual"
	webhookEventPath = ""
}

// Test status command
func TestStatusCmd_NoApp(t *testing.T) {
	resetFlags()
	cli.SetApp(nil)

	var output strings.Builder
	statusCmd.SetContext(context.Background())
	statusCmd.SetOut(&output)

	err := statusCmd.RunE(statusCmd, []string{})
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "requires database connection")
}

func TestStatusCmd_NoBillingService(t *testing.T) {
	resetFlags()
	app := &cli.App{
		CurrentUserID:  uuid.New(),
		BillingService: nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	var output strings.Builder
	statusCmd.SetContext(context.Background())
	statusCmd.SetOut(&output)

	err := statusCmd.RunE(statusCmd, []string{})
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "requires database connection")
}

// Test entitlements command
func TestEntitlementsCmd_NoApp(t *testing.T) {
	resetFlags()
	cli.SetApp(nil)

	var output strings.Builder
	entitlementsCmd.SetContext(context.Background())
	entitlementsCmd.SetOut(&output)

	err := entitlementsCmd.RunE(entitlementsCmd, []string{})
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "requires database connection")
}

func TestEntitlementsCmd_NoBillingService(t *testing.T) {
	resetFlags()
	app := &cli.App{
		CurrentUserID:  uuid.New(),
		BillingService: nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	var output strings.Builder
	entitlementsCmd.SetContext(context.Background())
	entitlementsCmd.SetOut(&output)

	err := entitlementsCmd.RunE(entitlementsCmd, []string{})
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "requires database connection")
}

// Test grant command
func TestGrantCmd_NoApp(t *testing.T) {
	resetFlags()
	cli.SetApp(nil)

	grantModule = "test-module"
	grantCmd.SetContext(context.Background())

	err := grantCmd.RunE(grantCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "require database connection")
}

func TestGrantCmd_NoBillingService(t *testing.T) {
	resetFlags()
	app := &cli.App{
		CurrentUserID:  uuid.New(),
		BillingService: nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	grantModule = "test-module"
	grantCmd.SetContext(context.Background())

	err := grantCmd.RunE(grantCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "require database connection")
}

func TestGrantCmd_MissingModuleNoBillingService(t *testing.T) {
	resetFlags()
	app := &cli.App{
		CurrentUserID:  uuid.New(),
		BillingService: nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	// Module validation happens after BillingService check
	// So we get database error first when BillingService is nil
	grantModule = ""
	grantCmd.SetContext(context.Background())

	err := grantCmd.RunE(grantCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "require database connection")
}

// Test webhook command
func TestWebhookCmd_MissingEventPath(t *testing.T) {
	resetFlags()
	webhookEventPath = ""

	webhookCmd.SetContext(context.Background())

	err := webhookCmd.RunE(webhookCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event path is required")
}

func TestWebhookCmd_NonexistentFile(t *testing.T) {
	resetFlags()
	webhookEventPath = "/nonexistent/path/to/event.json"

	webhookCmd.SetContext(context.Background())

	err := webhookCmd.RunE(webhookCmd, []string{})
	assert.Error(t, err)
}

func TestWebhookCmd_InvalidJSON(t *testing.T) {
	resetFlags()

	// Create a temp file with invalid JSON
	tmpDir, err := os.MkdirTemp("", "billing-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	eventFile := filepath.Join(tmpDir, "invalid.json")
	err = os.WriteFile(eventFile, []byte("not valid json"), 0644)
	require.NoError(t, err)

	webhookEventPath = eventFile
	webhookCmd.SetContext(context.Background())

	err = webhookCmd.RunE(webhookCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid webhook payload")
}

func TestWebhookCmd_ValidEvent(t *testing.T) {
	resetFlags()

	// Create a temp file with valid JSON
	tmpDir, err := os.MkdirTemp("", "billing-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	eventFile := filepath.Join(tmpDir, "event.json")
	eventJSON := `{"type": "customer.subscription.updated", "data": {"object": {}}}`
	err = os.WriteFile(eventFile, []byte(eventJSON), 0644)
	require.NoError(t, err)

	webhookEventPath = eventFile
	var output strings.Builder
	webhookCmd.SetContext(context.Background())
	webhookCmd.SetOut(&output)

	err = webhookCmd.RunE(webhookCmd, []string{})
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "customer.subscription.updated")
}

func TestWebhookCmd_UnknownEventType(t *testing.T) {
	resetFlags()

	// Create a temp file with JSON missing type field
	tmpDir, err := os.MkdirTemp("", "billing-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	eventFile := filepath.Join(tmpDir, "event.json")
	eventJSON := `{"data": {"object": {}}}`
	err = os.WriteFile(eventFile, []byte(eventJSON), 0644)
	require.NoError(t, err)

	webhookEventPath = eventFile
	var output strings.Builder
	webhookCmd.SetContext(context.Background())
	webhookCmd.SetOut(&output)

	err = webhookCmd.RunE(webhookCmd, []string{})
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "unknown")
}

// Test command configuration
func TestCmdConfiguration(t *testing.T) {
	assert.Equal(t, "billing", Cmd.Use)

	// Verify subcommands are registered
	subCmds := Cmd.Commands()
	cmdNames := make([]string, len(subCmds))
	for i, cmd := range subCmds {
		cmdNames[i] = cmd.Use
	}

	assert.Contains(t, cmdNames, "status")
	assert.Contains(t, cmdNames, "entitlements")
	assert.Contains(t, cmdNames, "grant")
	assert.Contains(t, cmdNames, "webhook")
}

func TestGrantCmdFlags(t *testing.T) {
	resetFlags()

	assert.NotNil(t, grantCmd.Flags().Lookup("module"))
	assert.NotNil(t, grantCmd.Flags().Lookup("active"))
	assert.NotNil(t, grantCmd.Flags().Lookup("source"))
}

func TestWebhookCmdFlags(t *testing.T) {
	resetFlags()

	assert.NotNil(t, webhookCmd.Flags().Lookup("event"))
}
