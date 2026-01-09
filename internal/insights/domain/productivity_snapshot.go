// Package domain contains the domain model for the insights bounded context.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ProductivitySnapshot represents a daily snapshot of productivity metrics.
type ProductivitySnapshot struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	SnapshotDate time.Time

	// Task metrics
	TasksCreated          int
	TasksCompleted        int
	TasksOverdue          int
	TaskCompletionRate    float64
	AvgTaskDurationMinutes int

	// Time block metrics
	BlocksScheduled     int
	BlocksCompleted     int
	BlocksMissed        int
	ScheduledMinutes    int
	CompletedMinutes    int
	BlockCompletionRate float64

	// Habit metrics
	HabitsDue            int
	HabitsCompleted      int
	HabitCompletionRate  float64
	LongestStreak        int

	// Focus metrics
	FocusSessions          int
	TotalFocusMinutes      int
	AvgFocusSessionMinutes int

	// Overall score
	ProductivityScore int

	// Breakdown data
	PeakHours      []PeakHour
	TimeByCategory map[string]int

	// Metadata
	ComputedAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// PeakHour represents productivity at a specific hour.
type PeakHour struct {
	Hour        int `json:"hour"`
	Completions int `json:"completions"`
}

// NewProductivitySnapshot creates a new productivity snapshot.
func NewProductivitySnapshot(userID uuid.UUID, date time.Time) *ProductivitySnapshot {
	now := time.Now()
	return &ProductivitySnapshot{
		ID:             uuid.New(),
		UserID:         userID,
		SnapshotDate:   date,
		PeakHours:      []PeakHour{},
		TimeByCategory: make(map[string]int),
		ComputedAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// CalculateProductivityScore computes the overall productivity score (0-100).
func (s *ProductivitySnapshot) CalculateProductivityScore() {
	var score float64
	var weights float64

	// Task completion contributes 30%
	if s.TasksCreated > 0 || s.TasksCompleted > 0 {
		taskScore := s.TaskCompletionRate * 0.3
		// Bonus for completing overdue tasks
		if s.TasksOverdue > 0 && s.TasksCompleted > s.TasksOverdue {
			taskScore *= 1.1
		}
		score += taskScore * 100
		weights += 0.3
	}

	// Time block completion contributes 30%
	if s.BlocksScheduled > 0 {
		blockScore := s.BlockCompletionRate * 0.3
		score += blockScore * 100
		weights += 0.3
	}

	// Habit completion contributes 25%
	if s.HabitsDue > 0 {
		habitScore := s.HabitCompletionRate * 0.25
		// Bonus for maintaining streaks
		if s.LongestStreak >= 7 {
			habitScore *= 1.1
		}
		score += habitScore * 100
		weights += 0.25
	}

	// Focus time contributes 15%
	if s.FocusSessions > 0 {
		// Assume 4 hours of focus time is excellent
		focusRatio := float64(s.TotalFocusMinutes) / 240.0
		if focusRatio > 1.0 {
			focusRatio = 1.0
		}
		focusScore := focusRatio * 0.15
		// Bonus for longer average sessions (>25 min pomodoro)
		if s.AvgFocusSessionMinutes >= 25 {
			focusScore *= 1.1
		}
		score += focusScore * 100
		weights += 0.15
	}

	// Normalize score if not all categories are present
	if weights > 0 {
		score = score / weights * (weights / 1.0)
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	s.ProductivityScore = int(score)
}

// SetTaskMetrics sets task-related metrics.
func (s *ProductivitySnapshot) SetTaskMetrics(created, completed, overdue int, avgDuration int) {
	s.TasksCreated = created
	s.TasksCompleted = completed
	s.TasksOverdue = overdue
	s.AvgTaskDurationMinutes = avgDuration

	total := created + completed // Tasks that could have been completed
	if total > 0 {
		s.TaskCompletionRate = float64(completed) / float64(total)
	}
}

// SetBlockMetrics sets time block-related metrics.
func (s *ProductivitySnapshot) SetBlockMetrics(scheduled, completed, missed, scheduledMins, completedMins int) {
	s.BlocksScheduled = scheduled
	s.BlocksCompleted = completed
	s.BlocksMissed = missed
	s.ScheduledMinutes = scheduledMins
	s.CompletedMinutes = completedMins

	if scheduled > 0 {
		s.BlockCompletionRate = float64(completed) / float64(scheduled)
	}
}

// SetHabitMetrics sets habit-related metrics.
func (s *ProductivitySnapshot) SetHabitMetrics(due, completed, longestStreak int) {
	s.HabitsDue = due
	s.HabitsCompleted = completed
	s.LongestStreak = longestStreak

	if due > 0 {
		s.HabitCompletionRate = float64(completed) / float64(due)
	}
}

// SetFocusMetrics sets focus session metrics.
func (s *ProductivitySnapshot) SetFocusMetrics(sessions, totalMinutes int) {
	s.FocusSessions = sessions
	s.TotalFocusMinutes = totalMinutes

	if sessions > 0 {
		s.AvgFocusSessionMinutes = totalMinutes / sessions
	}
}
