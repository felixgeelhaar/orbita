// Package pomodoro provides an example Pomodoro timer orbit for Orbita.
// This demonstrates how to build a third-party orbit using the public orbitsdk package.
package pomodoro

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sdk "github.com/felixgeelhaar/orbita/pkg/orbitsdk"
)

const (
	// OrbitID is the unique identifier for this orbit.
	OrbitID = "acme.pomodoro"

	// Default configuration values
	defaultWorkDuration     = 25 * time.Minute
	defaultShortBreak       = 5 * time.Minute
	defaultLongBreak        = 15 * time.Minute
	defaultSessionsForLong  = 4
)

// Config holds the configurable settings for the Pomodoro orbit.
type Config struct {
	WorkDuration          time.Duration `json:"work_duration"`
	ShortBreak            time.Duration `json:"short_break"`
	LongBreak             time.Duration `json:"long_break"`
	SessionsUntilLongBreak int          `json:"sessions_until_long_break"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		WorkDuration:          defaultWorkDuration,
		ShortBreak:            defaultShortBreak,
		LongBreak:             defaultLongBreak,
		SessionsUntilLongBreak: defaultSessionsForLong,
	}
}

// SessionState represents the current state of a Pomodoro session.
type SessionState struct {
	TaskID         string    `json:"task_id,omitempty"`
	TaskTitle      string    `json:"task_title,omitempty"`
	StartedAt      time.Time `json:"started_at"`
	EndsAt         time.Time `json:"ends_at"`
	SessionType    string    `json:"session_type"` // "work", "short_break", "long_break"
	SessionNumber  int       `json:"session_number"`
	TotalSessions  int       `json:"total_sessions_today"`
	IsActive       bool      `json:"is_active"`
}

// DailyStats tracks Pomodoro statistics for the day.
type DailyStats struct {
	Date              string   `json:"date"`
	CompletedSessions int      `json:"completed_sessions"`
	TotalWorkMinutes  int      `json:"total_work_minutes"`
	TasksWorkedOn     []string `json:"tasks_worked_on"`
}

// Orbit implements the Pomodoro timer orbit.
type Orbit struct {
	config  Config
	ctx     sdk.Context
	storage sdk.StorageAPI
}

// New creates a new Pomodoro orbit instance.
func New() *Orbit {
	return &Orbit{
		config: DefaultConfig(),
	}
}

// NewWithConfig creates a new Pomodoro orbit with custom configuration.
func NewWithConfig(cfg Config) *Orbit {
	return &Orbit{
		config: cfg,
	}
}

// Metadata returns the orbit's metadata.
func (o *Orbit) Metadata() sdk.Metadata {
	return sdk.Metadata{
		ID:            OrbitID,
		Name:          "ACME Pomodoro Timer",
		Version:       "1.0.0",
		Author:        "ACME Corp",
		Description:   "A Pomodoro technique timer that integrates with tasks and schedule",
		License:       "MIT",
		Homepage:      "https://acme.example.com/orbita-orbits/pomodoro",
		Tags:          []string{"productivity", "focus", "time-management"},
		MinAPIVersion: "1.0.0",
	}
}

// RequiredCapabilities returns the capabilities required by this orbit.
func (o *Orbit) RequiredCapabilities() []sdk.Capability {
	return []sdk.Capability{
		sdk.CapReadTasks,
		sdk.CapReadSchedule,
		sdk.CapReadStorage,
		sdk.CapWriteStorage,
		sdk.CapSubscribeEvents,
		sdk.CapRegisterTools,
	}
}

// Initialize sets up the orbit with its context.
func (o *Orbit) Initialize(ctx sdk.Context) error {
	o.ctx = ctx
	o.storage = ctx.Storage()

	ctx.Logger().Info("pomodoro orbit initialized",
		"orbit_id", OrbitID,
		"work_duration", o.config.WorkDuration,
	)

	return nil
}

// Shutdown cleans up the orbit.
func (o *Orbit) Shutdown(ctx context.Context) error {
	// Nothing to clean up for this simple orbit
	return nil
}

// RegisterTools registers MCP tools for the Pomodoro orbit.
func (o *Orbit) RegisterTools(registry sdk.ToolRegistry) error {
	// Register start_pomodoro tool
	if err := registry.RegisterTool("start_pomodoro", o.handleStartPomodoro, sdk.ToolSchema{
		Description: "Start a new Pomodoro work session, optionally linked to a task",
		Properties: map[string]sdk.PropertySchema{
			"task_id": {
				Type:        "string",
				Description: "Optional task ID to associate with this session",
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to register start_pomodoro: %w", err)
	}

	// Register stop_pomodoro tool
	if err := registry.RegisterTool("stop_pomodoro", o.handleStopPomodoro, sdk.ToolSchema{
		Description: "Stop the current Pomodoro session",
	}); err != nil {
		return fmt.Errorf("failed to register stop_pomodoro: %w", err)
	}

	// Register pomodoro_status tool
	if err := registry.RegisterTool("pomodoro_status", o.handlePomodoroStatus, sdk.ToolSchema{
		Description: "Get the current Pomodoro session status",
	}); err != nil {
		return fmt.Errorf("failed to register pomodoro_status: %w", err)
	}

	// Register pomodoro_stats tool
	if err := registry.RegisterTool("pomodoro_stats", o.handlePomodoroStats, sdk.ToolSchema{
		Description: "Get Pomodoro statistics for today",
	}); err != nil {
		return fmt.Errorf("failed to register pomodoro_stats: %w", err)
	}

	return nil
}

// RegisterCommands registers CLI commands for the orbit.
func (o *Orbit) RegisterCommands(registry sdk.CommandRegistry) error {
	// This example doesn't register CLI commands
	return nil
}

// SubscribeEvents subscribes to domain events.
func (o *Orbit) SubscribeEvents(bus sdk.EventBus) error {
	// Subscribe to task completion events to track Pomodoro sessions
	return bus.Subscribe("tasks.task.completed", o.handleTaskCompleted)
}

// handleStartPomodoro starts a new Pomodoro session.
func (o *Orbit) handleStartPomodoro(ctx context.Context, input map[string]any) (any, error) {
	// Check if there's already an active session
	state, err := o.getCurrentSession(ctx)
	if err == nil && state != nil && state.IsActive {
		return map[string]any{
			"success": false,
			"error":   "A Pomodoro session is already active",
			"session": state,
		}, nil
	}

	// Get task info if provided
	var taskTitle string
	taskID, _ := input["task_id"].(string)
	if taskID != "" && o.ctx != nil {
		tasks := o.ctx.Tasks()
		if tasks != nil {
			task, err := tasks.Get(ctx, taskID)
			if err == nil && task != nil {
				taskTitle = task.Title
			}
		}
	}

	// Get today's stats to determine session number
	stats, _ := o.getTodayStats(ctx)
	sessionNumber := 1
	if stats != nil {
		sessionNumber = stats.CompletedSessions + 1
	}

	// Determine session type (work or break)
	sessionType := "work"
	duration := o.config.WorkDuration

	// Create new session
	now := time.Now()
	session := &SessionState{
		TaskID:        taskID,
		TaskTitle:     taskTitle,
		StartedAt:     now,
		EndsAt:        now.Add(duration),
		SessionType:   sessionType,
		SessionNumber: sessionNumber,
		IsActive:      true,
	}

	// Save session state
	if err := o.saveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return map[string]any{
		"success":        true,
		"message":        fmt.Sprintf("Started Pomodoro session #%d", sessionNumber),
		"session":        session,
		"duration_min":   int(duration.Minutes()),
		"ends_at":        session.EndsAt.Format(time.RFC3339),
	}, nil
}

// handleStopPomodoro stops the current session.
func (o *Orbit) handleStopPomodoro(ctx context.Context, input map[string]any) (any, error) {
	session, err := o.getCurrentSession(ctx)
	if err != nil || session == nil || !session.IsActive {
		return map[string]any{
			"success": false,
			"error":   "No active Pomodoro session",
		}, nil
	}

	// Calculate how long the session ran
	elapsed := time.Since(session.StartedAt)
	completed := elapsed >= o.config.WorkDuration

	// Mark session as inactive
	session.IsActive = false

	// Update stats if it was a work session
	if session.SessionType == "work" && completed {
		stats, _ := o.getTodayStats(ctx)
		if stats == nil {
			stats = &DailyStats{
				Date:              time.Now().Format("2006-01-02"),
				TasksWorkedOn:     []string{},
			}
		}
		stats.CompletedSessions++
		stats.TotalWorkMinutes += int(elapsed.Minutes())
		if session.TaskID != "" && !contains(stats.TasksWorkedOn, session.TaskID) {
			stats.TasksWorkedOn = append(stats.TasksWorkedOn, session.TaskID)
		}
		if err := o.saveStats(ctx, stats); err != nil {
			// Log but don't fail
			o.ctx.Logger().Warn("failed to save stats", "error", err)
		}
	}

	// Clear session
	if err := o.clearSession(ctx); err != nil {
		return nil, fmt.Errorf("failed to clear session: %w", err)
	}

	// Determine next action
	nextAction := "short_break"
	if session.SessionNumber%o.config.SessionsUntilLongBreak == 0 {
		nextAction = "long_break"
	}

	return map[string]any{
		"success":       true,
		"completed":     completed,
		"elapsed_min":   int(elapsed.Minutes()),
		"next_action":   nextAction,
		"message":       fmt.Sprintf("Session stopped after %d minutes", int(elapsed.Minutes())),
	}, nil
}

// handlePomodoroStatus returns the current session status.
func (o *Orbit) handlePomodoroStatus(ctx context.Context, input map[string]any) (any, error) {
	session, err := o.getCurrentSession(ctx)
	if err != nil || session == nil {
		return map[string]any{
			"active":  false,
			"message": "No active Pomodoro session",
		}, nil
	}

	if !session.IsActive {
		return map[string]any{
			"active":  false,
			"message": "No active Pomodoro session",
		}, nil
	}

	remaining := time.Until(session.EndsAt)
	if remaining < 0 {
		remaining = 0
	}

	return map[string]any{
		"active":         true,
		"session":        session,
		"remaining_min":  int(remaining.Minutes()),
		"remaining_sec":  int(remaining.Seconds()) % 60,
		"progress_pct":   100 - int(remaining.Minutes()*100/o.config.WorkDuration.Minutes()),
	}, nil
}

// handlePomodoroStats returns today's statistics.
func (o *Orbit) handlePomodoroStats(ctx context.Context, input map[string]any) (any, error) {
	stats, err := o.getTodayStats(ctx)
	if err != nil || stats == nil {
		return map[string]any{
			"date":               time.Now().Format("2006-01-02"),
			"completed_sessions": 0,
			"total_work_minutes": 0,
			"tasks_worked_on":    []string{},
		}, nil
	}

	return stats, nil
}

// handleTaskCompleted handles task completion events.
func (o *Orbit) handleTaskCompleted(ctx context.Context, event sdk.DomainEvent) error {
	// If there's an active session for this task, stop it
	session, _ := o.getCurrentSession(ctx)
	if session != nil && session.IsActive {
		taskID, _ := event.Payload["task_id"].(string)
		if session.TaskID == taskID {
			o.ctx.Logger().Info("auto-stopping pomodoro session due to task completion",
				"task_id", taskID,
			)
			// The user completed the task, so we mark the session as done
			o.handleStopPomodoro(ctx, nil)
		}
	}
	return nil
}

// Storage key helpers
const (
	sessionKey = "current_session"
	statsKey   = "daily_stats"
)

func (o *Orbit) getCurrentSession(ctx context.Context) (*SessionState, error) {
	if o.storage == nil {
		return nil, fmt.Errorf("storage not available")
	}
	data, err := o.storage.Get(ctx, sessionKey)
	if err != nil {
		return nil, err
	}
	var session SessionState
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (o *Orbit) saveSession(ctx context.Context, session *SessionState) error {
	if o.storage == nil {
		return fmt.Errorf("storage not available")
	}
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return o.storage.Set(ctx, sessionKey, data, 24*time.Hour)
}

func (o *Orbit) clearSession(ctx context.Context) error {
	if o.storage == nil {
		return fmt.Errorf("storage not available")
	}
	return o.storage.Delete(ctx, sessionKey)
}

func (o *Orbit) getTodayStats(ctx context.Context) (*DailyStats, error) {
	if o.storage == nil {
		return nil, fmt.Errorf("storage not available")
	}
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("%s:%s", statsKey, today)
	data, err := o.storage.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var stats DailyStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (o *Orbit) saveStats(ctx context.Context, stats *DailyStats) error {
	if o.storage == nil {
		return fmt.Errorf("storage not available")
	}
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("%s:%s", statsKey, today)
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	// Stats expire after 30 days
	return o.storage.Set(ctx, key, data, 30*24*time.Hour)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
