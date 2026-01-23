package domain

import "time"

// RiskType represents the type of risk affecting a project.
type RiskType string

const (
	// RiskOverdueTasks indicates tasks are past their due date.
	RiskOverdueTasks RiskType = "overdue_tasks"
	// RiskBlockedMilestone indicates a milestone is blocked.
	RiskBlockedMilestone RiskType = "blocked_milestone"
	// RiskScopeCreep indicates scope has expanded beyond original plan.
	RiskScopeCreep RiskType = "scope_creep"
	// RiskNoProgress indicates no progress has been made recently.
	RiskNoProgress RiskType = "no_progress"
	// RiskMissingDeadline indicates the project may miss its deadline.
	RiskMissingDeadline RiskType = "missing_deadline"
	// RiskUnassignedTasks indicates tasks have no assignee.
	RiskUnassignedTasks RiskType = "unassigned_tasks"
	// RiskDependencyChain indicates a long dependency chain.
	RiskDependencyChain RiskType = "dependency_chain"
)

// String returns the string representation of the risk type.
func (r RiskType) String() string {
	return string(r)
}

// IsValid returns true if the risk type is a known value.
func (r RiskType) IsValid() bool {
	switch r {
	case RiskOverdueTasks, RiskBlockedMilestone, RiskScopeCreep, RiskNoProgress,
		RiskMissingDeadline, RiskUnassignedTasks, RiskDependencyChain:
		return true
	default:
		return false
	}
}

// Severity represents the severity level of a risk.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// String returns the string representation of the severity.
func (s Severity) String() string {
	return string(s)
}

// IsValid returns true if the severity is a known value.
func (s Severity) IsValid() bool {
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return true
	default:
		return false
	}
}

// Weight returns a numeric weight for the severity (used in health calculations).
func (s Severity) Weight() float64 {
	switch s {
	case SeverityLow:
		return 0.1
	case SeverityMedium:
		return 0.25
	case SeverityHigh:
		return 0.5
	case SeverityCritical:
		return 1.0
	default:
		return 0.0
	}
}

// RiskFactor represents a specific risk affecting a project.
type RiskFactor struct {
	Type        RiskType
	Severity    Severity
	Description string
	Suggestion  string
	DetectedAt  time.Time
}

// NewRiskFactor creates a new risk factor.
func NewRiskFactor(riskType RiskType, severity Severity, description, suggestion string) RiskFactor {
	return RiskFactor{
		Type:        riskType,
		Severity:    severity,
		Description: description,
		Suggestion:  suggestion,
		DetectedAt:  time.Now().UTC(),
	}
}

// HealthScore represents the overall health of a project.
type HealthScore struct {
	Overall     float64      // 0.0 - 1.0 (1.0 = perfect health)
	OnTrack     bool         // Whether project is likely to meet deadlines
	RiskFactors []RiskFactor // Current risk factors
	LastUpdated time.Time    // When health was last calculated
}

// NewHealthScore creates a new health score.
func NewHealthScore() HealthScore {
	return HealthScore{
		Overall:     1.0, // Start with perfect health
		OnTrack:     true,
		RiskFactors: []RiskFactor{},
		LastUpdated: time.Now().UTC(),
	}
}

// Percentage returns the health score as a percentage (0-100).
func (h HealthScore) Percentage() int {
	return int(h.Overall * 100)
}

// AddRiskFactor adds a risk factor and recalculates health.
func (h *HealthScore) AddRiskFactor(risk RiskFactor) {
	h.RiskFactors = append(h.RiskFactors, risk)
	h.recalculate()
}

// ClearRiskFactors removes all risk factors.
func (h *HealthScore) ClearRiskFactors() {
	h.RiskFactors = []RiskFactor{}
	h.recalculate()
}

// recalculate updates the health score based on risk factors.
func (h *HealthScore) recalculate() {
	if len(h.RiskFactors) == 0 {
		h.Overall = 1.0
		h.OnTrack = true
		h.LastUpdated = time.Now().UTC()
		return
	}

	// Calculate total risk weight
	totalWeight := 0.0
	hasHighOrCritical := false

	for _, risk := range h.RiskFactors {
		totalWeight += risk.Severity.Weight()
		if risk.Severity == SeverityHigh || risk.Severity == SeverityCritical {
			hasHighOrCritical = true
		}
	}

	// Cap the weight at 1.0 (0% health)
	if totalWeight > 1.0 {
		totalWeight = 1.0
	}

	h.Overall = 1.0 - totalWeight
	h.OnTrack = !hasHighOrCritical && h.Overall >= 0.6
	h.LastUpdated = time.Now().UTC()
}

// HighestSeverity returns the highest severity among risk factors.
func (h HealthScore) HighestSeverity() Severity {
	highest := SeverityLow
	severityOrder := map[Severity]int{
		SeverityLow:      0,
		SeverityMedium:   1,
		SeverityHigh:     2,
		SeverityCritical: 3,
	}

	for _, risk := range h.RiskFactors {
		if severityOrder[risk.Severity] > severityOrder[highest] {
			highest = risk.Severity
		}
	}

	return highest
}
