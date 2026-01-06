package services

import (
	"fmt"
	"math"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
)

// PriorityEngineConfig tunes how signals combine into a score.
type PriorityEngineConfig struct {
	PriorityWeight       float64
	DueWeight            float64
	EffortWeight         float64
	StreakRiskWeight     float64
	MeetingCadenceWeight float64
}

// DefaultPriorityEngineConfig returns a production-friendly configuration.
func DefaultPriorityEngineConfig() PriorityEngineConfig {
	return PriorityEngineConfig{
		PriorityWeight:       2.0,
		DueWeight:            3.0,
		EffortWeight:         1.5,
		StreakRiskWeight:     1.0,
		MeetingCadenceWeight: 0.8,
	}
}

// PrioritySignals contains the attributes that influence a task's score.
type PrioritySignals struct {
	Priority       value_objects.Priority
	DueDate        *time.Time
	Duration       value_objects.Duration
	StreakRisk     float64 // 0..1
	MeetingCadence float64 // 0..1
}

// PriorityEngine computes priority scores from multiple signals.
type PriorityEngine struct {
	config PriorityEngineConfig
}

// NewPriorityEngine creates a new engine with the given configuration.
func NewPriorityEngine(cfg PriorityEngineConfig) *PriorityEngine {
	return &PriorityEngine{config: cfg}
}

// Score computes a score and human-readable explanation for the provided signals.
func (e *PriorityEngine) Score(signals PrioritySignals) (float64, string) {
	priorityBase := float64(signals.Priority.Weight())
	if priorityBase == 0 {
		priorityBase = 0.5 // give a small baseline for "none"
	}

	dueFactor := e.dueScore(signals.DueDate)
	effortFactor := e.effortScore(signals.Duration)
	streak := clamp01(signals.StreakRisk)
	meeting := clamp01(signals.MeetingCadence)

	score := priorityBase*e.config.PriorityWeight +
		dueFactor*e.config.DueWeight +
		effortFactor*e.config.EffortWeight +
		streak*e.config.StreakRiskWeight +
		meeting*e.config.MeetingCadenceWeight

	score = math.Round(score*100) / 100 // keep two decimal places

	explanation := fmt.Sprintf(
		"priority=%.2f due=%.2f effort=%.2f streak=%.2f meeting=%.2f",
		priorityBase*e.config.PriorityWeight,
		dueFactor*e.config.DueWeight,
		effortFactor*e.config.EffortWeight,
		streak*e.config.StreakRiskWeight,
		meeting*e.config.MeetingCadenceWeight,
	)

	return score, explanation
}

func (e *PriorityEngine) dueScore(due *time.Time) float64 {
	if due == nil {
		return 0
	}
	now := time.Now()
	days := due.Sub(now).Hours() / 24
	if days < 0 {
		return 1
	}

	return clamp01((14.0 - days) / 14.0)
}

func (e *PriorityEngine) effortScore(duration value_objects.Duration) float64 {
	if duration.IsZero() {
		return 1
	}
	hours := duration.Hours()
	return clamp01(1 - (hours / 8.0))
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
