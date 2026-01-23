package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRiskType_String(t *testing.T) {
	tests := []struct {
		riskType RiskType
		expected string
	}{
		{RiskOverdueTasks, "overdue_tasks"},
		{RiskBlockedMilestone, "blocked_milestone"},
		{RiskScopeCreep, "scope_creep"},
		{RiskNoProgress, "no_progress"},
		{RiskMissingDeadline, "missing_deadline"},
		{RiskUnassignedTasks, "unassigned_tasks"},
		{RiskDependencyChain, "dependency_chain"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.riskType.String())
		})
	}
}

func TestRiskType_IsValid(t *testing.T) {
	validTypes := []RiskType{
		RiskOverdueTasks,
		RiskBlockedMilestone,
		RiskScopeCreep,
		RiskNoProgress,
		RiskMissingDeadline,
		RiskUnassignedTasks,
		RiskDependencyChain,
	}

	for _, rt := range validTypes {
		t.Run(string(rt), func(t *testing.T) {
			assert.True(t, rt.IsValid())
		})
	}

	assert.False(t, RiskType("invalid").IsValid())
	assert.False(t, RiskType("").IsValid())
}

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityLow, "low"},
		{SeverityMedium, "medium"},
		{SeverityHigh, "high"},
		{SeverityCritical, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.severity.String())
		})
	}
}

func TestSeverity_IsValid(t *testing.T) {
	validSeverities := []Severity{
		SeverityLow,
		SeverityMedium,
		SeverityHigh,
		SeverityCritical,
	}

	for _, s := range validSeverities {
		t.Run(string(s), func(t *testing.T) {
			assert.True(t, s.IsValid())
		})
	}

	assert.False(t, Severity("invalid").IsValid())
	assert.False(t, Severity("").IsValid())
}

func TestSeverity_Weight(t *testing.T) {
	tests := []struct {
		severity Severity
		expected float64
	}{
		{SeverityLow, 0.1},
		{SeverityMedium, 0.25},
		{SeverityHigh, 0.5},
		{SeverityCritical, 1.0},
		{Severity("unknown"), 0.0},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.severity.Weight())
		})
	}
}

func TestNewRiskFactor(t *testing.T) {
	beforeCreate := time.Now().UTC()
	risk := NewRiskFactor(RiskOverdueTasks, SeverityMedium, "2 tasks overdue", "Complete them")

	assert.Equal(t, RiskOverdueTasks, risk.Type)
	assert.Equal(t, SeverityMedium, risk.Severity)
	assert.Equal(t, "2 tasks overdue", risk.Description)
	assert.Equal(t, "Complete them", risk.Suggestion)
	assert.True(t, risk.DetectedAt.After(beforeCreate) || risk.DetectedAt.Equal(beforeCreate))
}

func TestNewHealthScore(t *testing.T) {
	health := NewHealthScore()

	assert.Equal(t, 1.0, health.Overall)
	assert.True(t, health.OnTrack)
	assert.Empty(t, health.RiskFactors)
	assert.False(t, health.LastUpdated.IsZero())
}

func TestHealthScore_Percentage(t *testing.T) {
	tests := []struct {
		overall  float64
		expected int
	}{
		{1.0, 100},
		{0.75, 75},
		{0.5, 50},
		{0.33, 33},
		{0.0, 0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			health := HealthScore{Overall: tt.overall}
			assert.Equal(t, tt.expected, health.Percentage())
		})
	}
}

func TestHealthScore_AddRiskFactor(t *testing.T) {
	health := NewHealthScore()

	risk := NewRiskFactor(RiskOverdueTasks, SeverityLow, "Minor delay", "Monitor")
	health.AddRiskFactor(risk)

	assert.Len(t, health.RiskFactors, 1)
	assert.Less(t, health.Overall, 1.0)
}

func TestHealthScore_AddRiskFactor_MultipleRisks(t *testing.T) {
	health := NewHealthScore()

	health.AddRiskFactor(NewRiskFactor(RiskOverdueTasks, SeverityLow, "", ""))
	health.AddRiskFactor(NewRiskFactor(RiskNoProgress, SeverityMedium, "", ""))

	assert.Len(t, health.RiskFactors, 2)
	// Overall should be 1.0 - 0.1 - 0.25 = 0.65
	assert.InDelta(t, 0.65, health.Overall, 0.01)
	assert.True(t, health.OnTrack) // No high or critical risks
}

func TestHealthScore_AddRiskFactor_HighRiskAffectsOnTrack(t *testing.T) {
	health := NewHealthScore()

	health.AddRiskFactor(NewRiskFactor(RiskMissingDeadline, SeverityHigh, "", ""))

	assert.False(t, health.OnTrack)
}

func TestHealthScore_AddRiskFactor_CriticalRiskAffectsOnTrack(t *testing.T) {
	health := NewHealthScore()

	health.AddRiskFactor(NewRiskFactor(RiskBlockedMilestone, SeverityCritical, "", ""))

	assert.False(t, health.OnTrack)
	assert.Equal(t, 0.0, health.Overall)
}

func TestHealthScore_AddRiskFactor_MaxCap(t *testing.T) {
	health := NewHealthScore()

	// Add multiple critical risks that would exceed 1.0
	health.AddRiskFactor(NewRiskFactor(RiskOverdueTasks, SeverityCritical, "", ""))
	health.AddRiskFactor(NewRiskFactor(RiskNoProgress, SeverityCritical, "", ""))

	// Overall should be capped at 0.0 (weight capped at 1.0)
	assert.Equal(t, 0.0, health.Overall)
}

func TestHealthScore_ClearRiskFactors(t *testing.T) {
	health := NewHealthScore()
	health.AddRiskFactor(NewRiskFactor(RiskOverdueTasks, SeverityHigh, "", ""))
	health.AddRiskFactor(NewRiskFactor(RiskNoProgress, SeverityMedium, "", ""))

	health.ClearRiskFactors()

	assert.Empty(t, health.RiskFactors)
	assert.Equal(t, 1.0, health.Overall)
	assert.True(t, health.OnTrack)
}

func TestHealthScore_HighestSeverity(t *testing.T) {
	tests := []struct {
		name       string
		severities []Severity
		expected   Severity
	}{
		{"no risks", nil, SeverityLow},
		{"single low", []Severity{SeverityLow}, SeverityLow},
		{"single high", []Severity{SeverityHigh}, SeverityHigh},
		{"mixed", []Severity{SeverityLow, SeverityHigh, SeverityMedium}, SeverityHigh},
		{"all critical", []Severity{SeverityCritical, SeverityCritical}, SeverityCritical},
		{"medium and high", []Severity{SeverityMedium, SeverityHigh}, SeverityHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := NewHealthScore()
			for _, s := range tt.severities {
				health.AddRiskFactor(NewRiskFactor(RiskOverdueTasks, s, "", ""))
			}

			assert.Equal(t, tt.expected, health.HighestSeverity())
		})
	}
}

func TestHealthScore_OnTrackThreshold(t *testing.T) {
	tests := []struct {
		name      string
		risks     []Severity
		onTrack   bool
	}{
		{"healthy", nil, true},
		{"just above threshold", []Severity{SeverityMedium}, true},                    // 1.0 - 0.25 = 0.75 >= 0.6
		{"at threshold", []Severity{SeverityLow, SeverityLow, SeverityLow}, true},     // 1.0 - 0.3 = 0.7 >= 0.6
		{"below threshold", []Severity{SeverityMedium, SeverityMedium}, false},        // 1.0 - 0.5 = 0.5 < 0.6
		{"high severity always off track", []Severity{SeverityHigh}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := NewHealthScore()
			for _, s := range tt.risks {
				health.AddRiskFactor(NewRiskFactor(RiskOverdueTasks, s, "", ""))
			}

			assert.Equal(t, tt.onTrack, health.OnTrack, "Overall: %f", health.Overall)
		})
	}
}
