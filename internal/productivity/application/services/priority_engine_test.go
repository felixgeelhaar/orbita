package services

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	"github.com/stretchr/testify/require"
)

func TestPriorityEngine_DueDateBoostsScore(t *testing.T) {
	engine := NewPriorityEngine(DefaultPriorityEngineConfig())

	soon := time.Now().Add(24 * time.Hour)
	far := time.Now().Add(14 * 24 * time.Hour)

	shortDuration := value_objects.MustNewDuration(30 * time.Minute)
	signalsSoon := PrioritySignals{
		Priority:       value_objects.PriorityHigh,
		DueDate:        &soon,
		Duration:       shortDuration,
		StreakRisk:     0.2,
		MeetingCadence: 0.1,
	}
	signalsFar := signalsSoon
	signalsFar.DueDate = &far

	scoreSoon, _ := engine.Score(signalsSoon)
	scoreFar, _ := engine.Score(signalsFar)

	require.Greater(t, scoreSoon, scoreFar, "tasks due sooner should score higher")
}

func TestPriorityEngine_ExplanationIncludesSignals(t *testing.T) {
	engine := NewPriorityEngine(DefaultPriorityEngineConfig())

	signals := PrioritySignals{
		Priority:       value_objects.PriorityUrgent,
		DueDate:        nil,
		Duration:       value_objects.MustNewDuration(15 * time.Minute),
		StreakRisk:     0.5,
		MeetingCadence: 0.3,
	}

	_, explanation := engine.Score(signals)
	require.Contains(t, explanation, "priority=")
	require.Contains(t, explanation, "streak=")
}
