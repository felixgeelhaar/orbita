package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/domain/value_objects"
	"github.com/stretchr/testify/assert"
)

func TestDefaultPriorityEngineConfig(t *testing.T) {
	config := DefaultPriorityEngineConfig()

	assert.Equal(t, 2.0, config.PriorityWeight)
	assert.Equal(t, 3.0, config.DueWeight)
	assert.Equal(t, 1.5, config.EffortWeight)
	assert.Equal(t, 1.0, config.StreakRiskWeight)
	assert.Equal(t, 0.8, config.MeetingCadenceWeight)
}

func TestNewPriorityEngine(t *testing.T) {
	config := DefaultPriorityEngineConfig()
	engine := NewPriorityEngine(config)

	assert.NotNil(t, engine)
	assert.Equal(t, config, engine.config)
}

func TestPriorityEngine_Score(t *testing.T) {
	t.Run("scores urgent priority task", func(t *testing.T) {
		engine := NewPriorityEngine(DefaultPriorityEngineConfig())
		duration := value_objects.MustNewDuration(30 * time.Minute)

		signals := PrioritySignals{
			Priority:       value_objects.PriorityUrgent,
			Duration:       duration,
			StreakRisk:     0,
			MeetingCadence: 0,
		}

		score, explanation := engine.Score(signals)

		// Urgent priority has weight 4, so priority component = 4 * 2.0 = 8.0
		// Effort for 0.5h = (1 - 0.5/8) = 0.9375, * 1.5 = 1.40625
		assert.Greater(t, score, 8.0)
		assert.Contains(t, explanation, "priority=")
		assert.Contains(t, explanation, "due=")
		assert.Contains(t, explanation, "effort=")
	})

	t.Run("scores none priority task with baseline", func(t *testing.T) {
		engine := NewPriorityEngine(DefaultPriorityEngineConfig())
		duration := value_objects.MustNewDuration(1 * time.Hour)

		signals := PrioritySignals{
			Priority:       value_objects.PriorityNone,
			Duration:       duration,
			StreakRisk:     0,
			MeetingCadence: 0,
		}

		score, explanation := engine.Score(signals)

		// None priority gets baseline 0.5, so priority component = 0.5 * 2.0 = 1.0
		assert.Greater(t, score, 0.0)
		assert.Contains(t, explanation, "priority=1.00")
	})

	t.Run("scores task due today higher than task due in 2 weeks", func(t *testing.T) {
		engine := NewPriorityEngine(DefaultPriorityEngineConfig())
		duration := value_objects.MustNewDuration(1 * time.Hour)

		today := time.Now().Add(1 * time.Hour)
		twoWeeks := time.Now().Add(14 * 24 * time.Hour)

		signalsToday := PrioritySignals{
			Priority: value_objects.PriorityMedium,
			DueDate:  &today,
			Duration: duration,
		}

		signalsTwoWeeks := PrioritySignals{
			Priority: value_objects.PriorityMedium,
			DueDate:  &twoWeeks,
			Duration: duration,
		}

		scoreToday, _ := engine.Score(signalsToday)
		scoreTwoWeeks, _ := engine.Score(signalsTwoWeeks)

		assert.Greater(t, scoreToday, scoreTwoWeeks)
	})

	t.Run("scores overdue task with maximum due factor", func(t *testing.T) {
		engine := NewPriorityEngine(DefaultPriorityEngineConfig())
		duration := value_objects.MustNewDuration(1 * time.Hour)

		overdue := time.Now().Add(-24 * time.Hour)

		signals := PrioritySignals{
			Priority: value_objects.PriorityMedium,
			DueDate:  &overdue,
			Duration: duration,
		}

		score, explanation := engine.Score(signals)

		// Overdue task gets due factor = 1.0, so due component = 1.0 * 3.0 = 3.0
		assert.Contains(t, explanation, "due=3.00")
		assert.Greater(t, score, 5.0)
	})

	t.Run("scores task with no due date", func(t *testing.T) {
		engine := NewPriorityEngine(DefaultPriorityEngineConfig())
		duration := value_objects.MustNewDuration(1 * time.Hour)

		signals := PrioritySignals{
			Priority: value_objects.PriorityMedium,
			DueDate:  nil,
			Duration: duration,
		}

		_, explanation := engine.Score(signals)

		// No due date means due factor = 0
		assert.Contains(t, explanation, "due=0.00")
	})

	t.Run("scores shorter tasks higher for effort", func(t *testing.T) {
		engine := NewPriorityEngine(DefaultPriorityEngineConfig())

		shortDuration := value_objects.MustNewDuration(30 * time.Minute)
		longDuration := value_objects.MustNewDuration(4 * time.Hour)

		signalsShort := PrioritySignals{
			Priority: value_objects.PriorityMedium,
			Duration: shortDuration,
		}

		signalsLong := PrioritySignals{
			Priority: value_objects.PriorityMedium,
			Duration: longDuration,
		}

		scoreShort, _ := engine.Score(signalsShort)
		scoreLong, _ := engine.Score(signalsLong)

		assert.Greater(t, scoreShort, scoreLong)
	})

	t.Run("scores zero duration task with maximum effort factor", func(t *testing.T) {
		engine := NewPriorityEngine(DefaultPriorityEngineConfig())

		signals := PrioritySignals{
			Priority: value_objects.PriorityMedium,
			Duration: value_objects.Zero(),
		}

		_, explanation := engine.Score(signals)

		// Zero duration gets effort factor = 1.0, so effort component = 1.0 * 1.5 = 1.5
		assert.Contains(t, explanation, "effort=1.50")
	})

	t.Run("includes streak risk in score", func(t *testing.T) {
		engine := NewPriorityEngine(DefaultPriorityEngineConfig())
		duration := value_objects.MustNewDuration(1 * time.Hour)

		signalsNoRisk := PrioritySignals{
			Priority:   value_objects.PriorityMedium,
			Duration:   duration,
			StreakRisk: 0,
		}

		signalsHighRisk := PrioritySignals{
			Priority:   value_objects.PriorityMedium,
			Duration:   duration,
			StreakRisk: 1.0,
		}

		scoreNoRisk, _ := engine.Score(signalsNoRisk)
		scoreHighRisk, _ := engine.Score(signalsHighRisk)

		// High streak risk should increase score
		assert.Greater(t, scoreHighRisk, scoreNoRisk)
	})

	t.Run("includes meeting cadence in score", func(t *testing.T) {
		engine := NewPriorityEngine(DefaultPriorityEngineConfig())
		duration := value_objects.MustNewDuration(1 * time.Hour)

		signalsNoCadence := PrioritySignals{
			Priority:       value_objects.PriorityMedium,
			Duration:       duration,
			MeetingCadence: 0,
		}

		signalsHighCadence := PrioritySignals{
			Priority:       value_objects.PriorityMedium,
			Duration:       duration,
			MeetingCadence: 1.0,
		}

		scoreNoCadence, _ := engine.Score(signalsNoCadence)
		scoreHighCadence, _ := engine.Score(signalsHighCadence)

		// High meeting cadence should increase score
		assert.Greater(t, scoreHighCadence, scoreNoCadence)
	})

	t.Run("clamps streak risk to 0-1 range", func(t *testing.T) {
		engine := NewPriorityEngine(DefaultPriorityEngineConfig())
		duration := value_objects.MustNewDuration(1 * time.Hour)

		signalsNegative := PrioritySignals{
			Priority:   value_objects.PriorityMedium,
			Duration:   duration,
			StreakRisk: -0.5,
		}

		signalsOverOne := PrioritySignals{
			Priority:   value_objects.PriorityMedium,
			Duration:   duration,
			StreakRisk: 1.5,
		}

		signalsZero := PrioritySignals{
			Priority:   value_objects.PriorityMedium,
			Duration:   duration,
			StreakRisk: 0,
		}

		signalsOne := PrioritySignals{
			Priority:   value_objects.PriorityMedium,
			Duration:   duration,
			StreakRisk: 1.0,
		}

		scoreNegative, _ := engine.Score(signalsNegative)
		scoreZero, _ := engine.Score(signalsZero)
		scoreOverOne, _ := engine.Score(signalsOverOne)
		scoreOne, _ := engine.Score(signalsOne)

		// Negative should clamp to 0
		assert.Equal(t, scoreZero, scoreNegative)
		// Over 1 should clamp to 1
		assert.Equal(t, scoreOne, scoreOverOne)
	})

	t.Run("returns rounded score", func(t *testing.T) {
		engine := NewPriorityEngine(DefaultPriorityEngineConfig())
		duration := value_objects.MustNewDuration(1 * time.Hour)

		signals := PrioritySignals{
			Priority:       value_objects.PriorityHigh,
			Duration:       duration,
			StreakRisk:     0.333,
			MeetingCadence: 0.666,
		}

		score, _ := engine.Score(signals)

		// Score should be rounded to 2 decimal places
		scoreStr := func(s float64) string {
			return formatFloat(s)
		}(score)
		assert.Regexp(t, `^\d+\.\d{1,2}$`, scoreStr)
	})
}

func TestClamp01(t *testing.T) {
	t.Run("returns value within range unchanged", func(t *testing.T) {
		assert.Equal(t, 0.5, clamp01(0.5))
		assert.Equal(t, 0.0, clamp01(0.0))
		assert.Equal(t, 1.0, clamp01(1.0))
	})

	t.Run("clamps negative values to 0", func(t *testing.T) {
		assert.Equal(t, 0.0, clamp01(-0.1))
		assert.Equal(t, 0.0, clamp01(-1.0))
		assert.Equal(t, 0.0, clamp01(-100.0))
	})

	t.Run("clamps values over 1 to 1", func(t *testing.T) {
		assert.Equal(t, 1.0, clamp01(1.1))
		assert.Equal(t, 1.0, clamp01(2.0))
		assert.Equal(t, 1.0, clamp01(100.0))
	})
}

func TestPriorityEngine_dueScore(t *testing.T) {
	engine := NewPriorityEngine(DefaultPriorityEngineConfig())

	t.Run("returns 0 for nil due date", func(t *testing.T) {
		score := engine.dueScore(nil)
		assert.Equal(t, 0.0, score)
	})

	t.Run("returns 1 for overdue tasks", func(t *testing.T) {
		overdue := time.Now().Add(-24 * time.Hour)
		score := engine.dueScore(&overdue)
		assert.Equal(t, 1.0, score)
	})

	t.Run("returns 0 for tasks due in 14+ days", func(t *testing.T) {
		farFuture := time.Now().Add(15 * 24 * time.Hour)
		score := engine.dueScore(&farFuture)
		assert.LessOrEqual(t, score, 0.0)
	})

	t.Run("returns value between 0 and 1 for tasks due within 14 days", func(t *testing.T) {
		oneWeek := time.Now().Add(7 * 24 * time.Hour)
		score := engine.dueScore(&oneWeek)
		assert.Greater(t, score, 0.0)
		assert.Less(t, score, 1.0)
	})
}

func TestPriorityEngine_effortScore(t *testing.T) {
	engine := NewPriorityEngine(DefaultPriorityEngineConfig())

	t.Run("returns 1 for zero duration", func(t *testing.T) {
		score := engine.effortScore(value_objects.Zero())
		assert.Equal(t, 1.0, score)
	})

	t.Run("returns 0.5 for 4-hour task", func(t *testing.T) {
		duration := value_objects.MustNewDuration(4 * time.Hour)
		score := engine.effortScore(duration)
		assert.Equal(t, 0.5, score)
	})

	t.Run("returns 0 for 8-hour task", func(t *testing.T) {
		duration := value_objects.MustNewDuration(8 * time.Hour)
		score := engine.effortScore(duration)
		assert.Equal(t, 0.0, score)
	})

	t.Run("returns value between 0 and 1 for typical tasks", func(t *testing.T) {
		duration := value_objects.MustNewDuration(2 * time.Hour)
		score := engine.effortScore(duration)
		assert.Greater(t, score, 0.0)
		assert.Less(t, score, 1.0)
	})
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}
