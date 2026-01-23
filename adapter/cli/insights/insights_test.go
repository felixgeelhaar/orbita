package insights

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func resetFlags() {
	// Trends flags
	trendsDays = 14

	// Compute flags
	computeDate = ""

	// Session flags
	sessionType = "focus"
	sessionTitle = ""
	sessionCategory = ""
	sessionNotes = ""

	// Goal flags
	goalType = "daily_tasks"
	goalTarget = 0
	goalPeriod = "daily"
	goalListLimit = 10
	goalShowAll = false
}

// Test dashboard command
func TestDashboardCmd_NoService(t *testing.T) {
	resetFlags()
	SetService(nil)

	dashboardCmd.SetContext(context.Background())

	err := dashboardCmd.RunE(dashboardCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insights service not available")
}

// Test trends command
func TestTrendsCmd_NoService(t *testing.T) {
	resetFlags()
	SetService(nil)

	trendsCmd.SetContext(context.Background())

	err := trendsCmd.RunE(trendsCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insights service not available")
}

// Test compute command
func TestComputeCmd_NoService(t *testing.T) {
	resetFlags()
	SetService(nil)

	computeCmd.SetContext(context.Background())

	err := computeCmd.RunE(computeCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insights service not available")
}

func TestComputeCmd_InvalidDateFormat(t *testing.T) {
	resetFlags()
	SetService(nil)

	computeDate = "invalid-date"
	computeCmd.SetContext(context.Background())

	// Still fails on service check first - the date parsing happens after
	err := computeCmd.RunE(computeCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insights service not available")
}

// Test session start command
func TestSessionStartCmd_NoService(t *testing.T) {
	resetFlags()
	SetService(nil)

	sessionStartCmd.SetContext(context.Background())

	err := sessionStartCmd.RunE(sessionStartCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insights service not available")
}

// Test session end command
func TestSessionEndCmd_NoService(t *testing.T) {
	resetFlags()
	SetService(nil)

	sessionEndCmd.SetContext(context.Background())

	err := sessionEndCmd.RunE(sessionEndCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insights service not available")
}

// Test session status command
func TestSessionStatusCmd_NoService(t *testing.T) {
	resetFlags()
	SetService(nil)

	sessionStatusCmd.SetContext(context.Background())

	err := sessionStatusCmd.RunE(sessionStatusCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insights service not available")
}

// Test goal create command
func TestGoalCreateCmd_NoService(t *testing.T) {
	resetFlags()
	SetService(nil)

	goalTarget = 5 // Set a valid target
	goalCreateCmd.SetContext(context.Background())

	err := goalCreateCmd.RunE(goalCreateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insights service not available")
}

func TestGoalCreateCmd_InvalidTargetNoService(t *testing.T) {
	resetFlags()
	SetService(nil)

	// Service check happens before target validation
	// So we get service error first when insightsService is nil
	goalTarget = 0 // Invalid target (must be positive)
	goalCreateCmd.SetContext(context.Background())

	err := goalCreateCmd.RunE(goalCreateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insights service not available")
}

func TestGoalCreateCmd_NegativeTargetNoService(t *testing.T) {
	resetFlags()
	SetService(nil)

	// Service check happens before target validation
	// So we get service error first when insightsService is nil
	goalTarget = -5 // Invalid target (must be positive)
	goalCreateCmd.SetContext(context.Background())

	err := goalCreateCmd.RunE(goalCreateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insights service not available")
}

// Test goal list command
func TestGoalListCmd_NoService(t *testing.T) {
	resetFlags()
	SetService(nil)

	goalListCmd.SetContext(context.Background())

	err := goalListCmd.RunE(goalListCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insights service not available")
}

// Test goal achieved command
func TestGoalAchievedCmd_NoService(t *testing.T) {
	resetFlags()
	SetService(nil)

	goalAchievedCmd.SetContext(context.Background())

	err := goalAchievedCmd.RunE(goalAchievedCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insights service not available")
}

// Test helper functions
func TestProgressBar(t *testing.T) {
	tests := []struct {
		pct      float64
		width    int
		expected string
	}{
		{0, 10, "----------"},
		{50, 10, "=====-----"},
		{100, 10, "=========="},
		{120, 10, "==========" /* capped at 100 */},
		{25, 20, "=====---------------"},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			result := progressBar(tc.pct, tc.width)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTrendSign(t *testing.T) {
	tests := []struct {
		val      float64
		expected string
	}{
		{10.5, "+"},
		{0, "+"},
		{-5.5, ""},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			result := trendSign(tc.val)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Test command configuration
func TestCmdConfiguration(t *testing.T) {
	assert.Equal(t, "insights", Cmd.Use)

	// Verify subcommands are registered
	subCmds := Cmd.Commands()
	cmdNames := make([]string, len(subCmds))
	for i, cmd := range subCmds {
		cmdNames[i] = cmd.Use
	}

	assert.Contains(t, cmdNames, "dashboard")
	assert.Contains(t, cmdNames, "trends")
	assert.Contains(t, cmdNames, "session")
	assert.Contains(t, cmdNames, "goal")
	assert.Contains(t, cmdNames, "compute")
}

func TestDashboardCmdAliases(t *testing.T) {
	assert.Equal(t, "dashboard", dashboardCmd.Use)
	assert.Equal(t, []string{"dash", "d"}, dashboardCmd.Aliases)
}

func TestTrendsCmdAliases(t *testing.T) {
	assert.Equal(t, "trends", trendsCmd.Use)
	assert.Equal(t, []string{"trend", "t"}, trendsCmd.Aliases)
}

func TestComputeCmdAliases(t *testing.T) {
	assert.Equal(t, "compute", computeCmd.Use)
	assert.Equal(t, []string{"refresh", "sync"}, computeCmd.Aliases)
}

// Test session subcommands
func TestSessionCmdSubcommands(t *testing.T) {
	subCmds := sessionCmd.Commands()
	cmdNames := make([]string, len(subCmds))
	for i, cmd := range subCmds {
		cmdNames[i] = cmd.Use
	}

	assert.Contains(t, cmdNames, "start")
	assert.Contains(t, cmdNames, "end")
	assert.Contains(t, cmdNames, "status")
}

// Test goal subcommands
func TestGoalCmdSubcommands(t *testing.T) {
	subCmds := goalCmd.Commands()
	cmdNames := make([]string, len(subCmds))
	for i, cmd := range subCmds {
		cmdNames[i] = cmd.Use
	}

	assert.Contains(t, cmdNames, "create")
	assert.Contains(t, cmdNames, "list")
	assert.Contains(t, cmdNames, "achieved")
}

// Test command flags
func TestTrendsCmdFlags(t *testing.T) {
	resetFlags()

	assert.NotNil(t, trendsCmd.Flags().Lookup("days"))
}

func TestComputeCmdFlags(t *testing.T) {
	resetFlags()

	assert.NotNil(t, computeCmd.Flags().Lookup("date"))
}

func TestSessionStartCmdFlags(t *testing.T) {
	resetFlags()

	assert.NotNil(t, sessionStartCmd.Flags().Lookup("title"))
	assert.NotNil(t, sessionStartCmd.Flags().Lookup("type"))
	assert.NotNil(t, sessionStartCmd.Flags().Lookup("category"))
}

func TestSessionEndCmdFlags(t *testing.T) {
	resetFlags()

	assert.NotNil(t, sessionEndCmd.Flags().Lookup("notes"))
}

func TestGoalCreateCmdFlags(t *testing.T) {
	resetFlags()

	assert.NotNil(t, goalCreateCmd.Flags().Lookup("type"))
	assert.NotNil(t, goalCreateCmd.Flags().Lookup("target"))
	assert.NotNil(t, goalCreateCmd.Flags().Lookup("period"))
}

func TestGoalListCmdFlags(t *testing.T) {
	resetFlags()

	assert.NotNil(t, goalListCmd.Flags().Lookup("all"))
}

func TestGoalAchievedCmdFlags(t *testing.T) {
	resetFlags()

	assert.NotNil(t, goalAchievedCmd.Flags().Lookup("limit"))
}
