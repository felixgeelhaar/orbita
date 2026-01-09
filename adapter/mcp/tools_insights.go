package mcp

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	schedQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
)

// TimeSpentDTO represents time spent analysis.
type TimeSpentDTO struct {
	Period      string               `json:"period"`
	StartDate   string               `json:"start_date"`
	EndDate     string               `json:"end_date"`
	TotalHours  float64              `json:"total_hours"`
	ByCategory  map[string]float64   `json:"by_category"`
	ByDay       map[string]float64   `json:"by_day"`
	TopTasks    []TaskTimeDTO        `json:"top_tasks,omitempty"`
	Comparisons *ComparisonDTO       `json:"comparisons,omitempty"`
}

// TaskTimeDTO represents time spent on a task.
type TaskTimeDTO struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Hours    float64 `json:"hours"`
	Category string  `json:"category"`
}

// ComparisonDTO represents period-over-period comparison.
type ComparisonDTO struct {
	PreviousPeriodHours float64 `json:"previous_period_hours"`
	ChangePercent       float64 `json:"change_percent"`
	Trend               string  `json:"trend"` // "up", "down", "stable"
}

// ProductivityScoreDTO represents productivity metrics.
type ProductivityScoreDTO struct {
	Date              string             `json:"date"`
	Score             int                `json:"score"` // 0-100
	TasksCompleted    int                `json:"tasks_completed"`
	TasksPlanned      int                `json:"tasks_planned"`
	HabitsCompleted   int                `json:"habits_completed"`
	HabitsPlanned     int                `json:"habits_planned"`
	MeetingsHeld      int                `json:"meetings_held"`
	MeetingsPlanned   int                `json:"meetings_planned"`
	FocusTimeMinutes  int                `json:"focus_time_minutes"`
	ScheduleAdherence float64            `json:"schedule_adherence"` // 0.0-1.0
	Breakdown         map[string]int     `json:"breakdown"`
	Recommendations   []string           `json:"recommendations,omitempty"`
}

// TrendDTO represents trends over time.
type TrendDTO struct {
	Period     string       `json:"period"`
	DataPoints []DataPoint  `json:"data_points"`
	Average    float64      `json:"average"`
	Trend      string       `json:"trend"` // "improving", "declining", "stable"
	BestDay    string       `json:"best_day,omitempty"`
	WorstDay   string       `json:"worst_day,omitempty"`
}

// DataPoint represents a single data point in a trend.
type DataPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type insightsTimeSpentInput struct {
	Period    string `json:"period,omitempty"`     // "day", "week", "month"
	StartDate string `json:"start_date,omitempty"` // YYYY-MM-DD
	EndDate   string `json:"end_date,omitempty"`   // YYYY-MM-DD
}

type insightsScoreInput struct {
	Date string `json:"date,omitempty"` // YYYY-MM-DD, defaults to today
}

type insightsTrendsInput struct {
	Metric string `json:"metric" jsonschema:"required"` // "productivity", "focus", "completion"
	Period string `json:"period,omitempty"`             // "week", "month", "quarter"
}

type insightsSummaryInput struct {
	Period string `json:"period,omitempty"` // "day", "week", "month"
}

func registerInsightsTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("insights.time_spent").
		Description("Analyze time spent across tasks, habits, and meetings").
		Handler(func(ctx context.Context, input insightsTimeSpentInput) (*TimeSpentDTO, error) {
			if app == nil || app.GetScheduleHandler == nil {
				return nil, errors.New("insights requires database connection")
			}

			// Determine date range
			endDate := time.Now()
			var startDate time.Time
			period := input.Period
			if period == "" {
				period = "week"
			}

			switch period {
			case "day":
				startDate = endDate
			case "week":
				startDate = endDate.AddDate(0, 0, -7)
			case "month":
				startDate = endDate.AddDate(0, -1, 0)
			default:
				startDate = endDate.AddDate(0, 0, -7)
			}

			if input.StartDate != "" {
				parsed, err := time.Parse(dateLayout, input.StartDate)
				if err == nil {
					startDate = parsed
				}
			}
			if input.EndDate != "" {
				parsed, err := time.Parse(dateLayout, input.EndDate)
				if err == nil {
					endDate = parsed
				}
			}

			// Aggregate time from schedules
			byCategory := make(map[string]float64)
			byDay := make(map[string]float64)
			var totalMinutes float64

			for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
				schedule, err := app.GetScheduleHandler.Handle(ctx, schedQueries.GetScheduleQuery{
					UserID: app.CurrentUserID,
					Date:   d,
				})
				if err != nil {
					continue
				}

				dayKey := d.Format(dateLayout)
				var dayMinutes float64

				for _, block := range schedule.Blocks {
					if block.Completed {
						duration := float64(block.DurationMin)
						totalMinutes += duration
						dayMinutes += duration
						byCategory[block.BlockType] += duration / 60.0
					}
				}

				byDay[dayKey] = dayMinutes / 60.0
			}

			return &TimeSpentDTO{
				Period:     period,
				StartDate:  startDate.Format(dateLayout),
				EndDate:    endDate.Format(dateLayout),
				TotalHours: totalMinutes / 60.0,
				ByCategory: byCategory,
				ByDay:      byDay,
			}, nil
		})

	srv.Tool("insights.productivity_score").
		Description("Get productivity score for a specific date").
		Handler(func(ctx context.Context, input insightsScoreInput) (*ProductivityScoreDTO, error) {
			if app == nil || app.GetScheduleHandler == nil {
				return nil, errors.New("insights requires database connection")
			}

			date := time.Now()
			if input.Date != "" {
				parsed, err := time.Parse(dateLayout, input.Date)
				if err != nil {
					return nil, err
				}
				date = parsed
			}

			schedule, err := app.GetScheduleHandler.Handle(ctx, schedQueries.GetScheduleQuery{
				UserID: app.CurrentUserID,
				Date:   date,
			})
			if err != nil {
				return nil, err
			}

			// Calculate metrics from schedule
			var tasksCompleted, tasksPlanned int
			var habitsCompleted, habitsPlanned int
			var meetingsHeld, meetingsPlanned int
			var focusMinutes int
			var completedBlocks, totalBlocks int

			for _, block := range schedule.Blocks {
				totalBlocks++
				if block.Completed {
					completedBlocks++
				}

				switch block.BlockType {
				case "task":
					tasksPlanned++
					if block.Completed {
						tasksCompleted++
					}
				case "habit":
					habitsPlanned++
					if block.Completed {
						habitsCompleted++
					}
				case "meeting":
					meetingsPlanned++
					if block.Completed {
						meetingsHeld++
					}
				case "focus":
					if block.Completed {
						focusMinutes += block.DurationMin
					}
				}
			}

			// Calculate adherence
			var adherence float64
			if totalBlocks > 0 {
				adherence = float64(completedBlocks) / float64(totalBlocks)
			}

			// Calculate score (weighted)
			score := calculateProductivityScore(
				tasksCompleted, tasksPlanned,
				habitsCompleted, habitsPlanned,
				meetingsHeld, meetingsPlanned,
				adherence,
			)

			// Generate recommendations
			var recommendations []string
			if adherence < 0.5 {
				recommendations = append(recommendations, "Consider reducing scheduled items to improve adherence")
			}
			if focusMinutes < 60 {
				recommendations = append(recommendations, "Schedule more focus time for deep work")
			}
			if habitsPlanned == 0 {
				recommendations = append(recommendations, "Add habits to build consistent routines")
			}

			return &ProductivityScoreDTO{
				Date:              date.Format(dateLayout),
				Score:             score,
				TasksCompleted:    tasksCompleted,
				TasksPlanned:      tasksPlanned,
				HabitsCompleted:   habitsCompleted,
				HabitsPlanned:     habitsPlanned,
				MeetingsHeld:      meetingsHeld,
				MeetingsPlanned:   meetingsPlanned,
				FocusTimeMinutes:  focusMinutes,
				ScheduleAdherence: adherence,
				Breakdown: map[string]int{
					"tasks":    int(float64(tasksCompleted) / max(float64(tasksPlanned), 1) * 25),
					"habits":   int(float64(habitsCompleted) / max(float64(habitsPlanned), 1) * 25),
					"meetings": int(float64(meetingsHeld) / max(float64(meetingsPlanned), 1) * 25),
					"schedule": int(adherence * 25),
				},
				Recommendations: recommendations,
			}, nil
		})

	srv.Tool("insights.trends").
		Description("Get productivity trends over time").
		Handler(func(ctx context.Context, input insightsTrendsInput) (*TrendDTO, error) {
			if app == nil || app.GetScheduleHandler == nil {
				return nil, errors.New("insights requires database connection")
			}

			period := input.Period
			if period == "" {
				period = "week"
			}

			var days int
			switch period {
			case "week":
				days = 7
			case "month":
				days = 30
			case "quarter":
				days = 90
			default:
				days = 7
			}

			endDate := time.Now()
			startDate := endDate.AddDate(0, 0, -days)

			var dataPoints []DataPoint
			var sum float64
			var bestValue float64
			var worstValue float64 = 100
			var bestDay, worstDay string

			for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
				schedule, err := app.GetScheduleHandler.Handle(ctx, schedQueries.GetScheduleQuery{
					UserID: app.CurrentUserID,
					Date:   d,
				})
				if err != nil {
					continue
				}

				var value float64
				switch input.Metric {
				case "productivity":
					var completed, total int
					for _, b := range schedule.Blocks {
						total++
						if b.Completed {
							completed++
						}
					}
					if total > 0 {
						value = float64(completed) / float64(total) * 100
					}
				case "focus":
					for _, b := range schedule.Blocks {
						if b.BlockType == "focus" && b.Completed {
							value += float64(b.DurationMin)
						}
					}
				case "completion":
					var completed int
					for _, b := range schedule.Blocks {
						if b.Completed {
							completed++
						}
					}
					value = float64(completed)
				default:
					value = 0
				}

				dateKey := d.Format(dateLayout)
				dataPoints = append(dataPoints, DataPoint{Date: dateKey, Value: value})
				sum += value

				if value > bestValue {
					bestValue = value
					bestDay = dateKey
				}
				if value < worstValue {
					worstValue = value
					worstDay = dateKey
				}
			}

			avg := sum / float64(len(dataPoints))

			// Determine trend
			var trend string
			if len(dataPoints) >= 2 {
				recent := dataPoints[len(dataPoints)-1].Value
				older := dataPoints[0].Value
				if recent > older*1.1 {
					trend = "improving"
				} else if recent < older*0.9 {
					trend = "declining"
				} else {
					trend = "stable"
				}
			} else {
				trend = "stable"
			}

			return &TrendDTO{
				Period:     period,
				DataPoints: dataPoints,
				Average:    avg,
				Trend:      trend,
				BestDay:    bestDay,
				WorstDay:   worstDay,
			}, nil
		})

	srv.Tool("insights.summary").
		Description("Get a summary of productivity insights").
		Handler(func(ctx context.Context, input insightsSummaryInput) (map[string]any, error) {
			if app == nil || app.GetScheduleHandler == nil || app.ListTasksHandler == nil {
				return nil, errors.New("insights requires database connection")
			}

			period := input.Period
			if period == "" {
				period = "week"
			}

			// Get today's score
			todayScore := &ProductivityScoreDTO{Score: 0}
			schedule, _ := app.GetScheduleHandler.Handle(ctx, schedQueries.GetScheduleQuery{
				UserID: app.CurrentUserID,
				Date:   time.Now(),
			})
			if schedule != nil {
				var completed, total int
				for _, b := range schedule.Blocks {
					total++
					if b.Completed {
						completed++
					}
				}
				if total > 0 {
					todayScore.Score = int(float64(completed) / float64(total) * 100)
				}
			}

			// Get pending tasks count
			tasks, _ := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
				UserID: app.CurrentUserID,
				Status: "pending",
			})
			pendingTasks := len(tasks)

			// Get overdue tasks
			overdueTasks, _ := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
				UserID:  app.CurrentUserID,
				Overdue: true,
			})

			return map[string]any{
				"period":              period,
				"today_score":         todayScore.Score,
				"pending_tasks":       pendingTasks,
				"overdue_tasks":       len(overdueTasks),
				"streak_days":         0, // TODO: implement streak tracking
				"top_recommendation":  "Complete your highest priority task first",
			}, nil
		})

	return nil
}

func calculateProductivityScore(
	tasksCompleted, tasksPlanned,
	habitsCompleted, habitsPlanned,
	meetingsHeld, meetingsPlanned int,
	adherence float64,
) int {
	var score float64

	// Tasks: 30% weight
	if tasksPlanned > 0 {
		score += (float64(tasksCompleted) / float64(tasksPlanned)) * 30
	} else {
		score += 30 // Full score if nothing planned
	}

	// Habits: 25% weight
	if habitsPlanned > 0 {
		score += (float64(habitsCompleted) / float64(habitsPlanned)) * 25
	} else {
		score += 25
	}

	// Meetings: 20% weight
	if meetingsPlanned > 0 {
		score += (float64(meetingsHeld) / float64(meetingsPlanned)) * 20
	} else {
		score += 20
	}

	// Schedule adherence: 25% weight
	score += adherence * 25

	return int(score)
}
