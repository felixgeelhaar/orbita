package license

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func resetFlags() {
	deactivateForce = false
}

// Test status command
func TestStatusCmd_NoService(t *testing.T) {
	resetFlags()
	SetLicenseService(nil)

	statusCmd.SetContext(context.Background())

	err := statusCmd.RunE(statusCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "license service not available")
}

// Test activate command
func TestActivateCmd_NoService(t *testing.T) {
	resetFlags()
	SetLicenseService(nil)

	activateCmd.SetContext(context.Background())

	err := activateCmd.RunE(activateCmd, []string{"ORB-TEST-KEYS-HERE"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "license service not available")
}

func TestActivateCmd_InvalidKeyFormat(t *testing.T) {
	resetFlags()
	SetLicenseService(nil)

	activateCmd.SetContext(context.Background())

	// Service check happens first, so we get service error
	err := activateCmd.RunE(activateCmd, []string{"invalid-key"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "license service not available")
}

// Test deactivate command
func TestDeactivateCmd_NoService(t *testing.T) {
	resetFlags()
	SetLicenseService(nil)

	deactivateCmd.SetContext(context.Background())

	err := deactivateCmd.RunE(deactivateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "license service not available")
}

func TestDeactivateCmd_ForceFlag(t *testing.T) {
	resetFlags()
	SetLicenseService(nil)

	deactivateForce = true
	deactivateCmd.SetContext(context.Background())

	// Still fails on service check
	err := deactivateCmd.RunE(deactivateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "license service not available")
}

// Test upgrade command - this one doesn't require a service
// Note: We skip the actual execution because it opens a browser
func TestUpgradeCmd_Configuration(t *testing.T) {
	resetFlags()
	SetLicenseService(nil)

	// Just verify the command is configured correctly
	assert.Equal(t, "upgrade", UpgradeCmd.Use)
	assert.NotNil(t, UpgradeCmd.RunE)
	assert.Contains(t, UpgradeCmd.Long, "checkout")
}

// Test helper functions
func TestIsValidLicenseKeyFormat(t *testing.T) {
	// Pattern is [A-Z2-9]{4} - only uppercase A-Z and digits 2-9
	tests := []struct {
		key      string
		expected bool
	}{
		{"ORB-ABCD-EFGH-IJKL", true},
		{"ORB-2345-6789-ABCD", true},     // Digits 2-9 allowed
		{"ORB-AB23-CD45-EF67", true},     // Mix of letters and digits 2-9
		{"ORB-abcd-efgh-ijkl", false},    // Lowercase not allowed
		{"orb-ABCD-EFGH-IJKL", false},    // Lowercase prefix not allowed
		{"ABC-DEFG-HIJK-LMNO", false},    // Wrong prefix
		{"ORB-ABC-DEF-GHI", false},       // Too short segments
		{"ORB-ABCDE-FGHIJ-KLMNO", false}, // Too long segments
		{"ORBABCDEFGHIJKL", false},       // No dashes
		{"", false},                      // Empty
		{"ORB-0000-1111-AAAA", false},    // 0 and 1 not allowed, only 2-9
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			result := isValidLicenseKeyFormat(tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatEntitlementName(t *testing.T) {
	tests := []struct {
		module   string
		expected string
	}{
		{"smart-habits", "Smart Habits"},
		{"smart-1to1", "Smart 1:1 Scheduler"},
		{"auto-rescheduler", "Auto-Rescheduler"},
		{"ai-inbox", "AI Inbox"},
		{"priority-engine", "Priority Engine"},
		{"adaptive-frequency", "Adaptive Frequency"},
		{"unknown-module", "unknown-module"}, // Returns as-is if not found
	}

	for _, tc := range tests {
		t.Run(tc.module, func(t *testing.T) {
			result := formatEntitlementName(tc.module)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Test command configuration
func TestCmdConfiguration(t *testing.T) {
	assert.Equal(t, "license", Cmd.Use)

	// Verify subcommands are registered
	subCmds := Cmd.Commands()
	cmdNames := make([]string, len(subCmds))
	for i, cmd := range subCmds {
		cmdNames[i] = cmd.Use
	}

	assert.Contains(t, cmdNames, "status")
	assert.Contains(t, cmdNames, "activate <license-key>")
	assert.Contains(t, cmdNames, "deactivate")
}

func TestUpgradeCmdConfiguration(t *testing.T) {
	assert.Equal(t, "upgrade", UpgradeCmd.Use)
}

func TestActivateCmdRequiresExactlyOneArg(t *testing.T) {
	assert.NotNil(t, activateCmd.Args)
}

// Test command flags
func TestDeactivateCmdFlags(t *testing.T) {
	resetFlags()

	assert.NotNil(t, deactivateCmd.Flags().Lookup("force"))
}

// Test openBrowser function signature exists
// Note: We don't actually call openBrowser because it opens real browsers
func TestOpenBrowserExists(t *testing.T) {
	// Just verify the function exists and has the expected signature
	// We can't test it without opening real browsers
	var fn func(string) bool = openBrowser
	assert.NotNil(t, fn)
}
