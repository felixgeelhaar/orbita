package mcp

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/felixgeelhaar/mcp-go"
	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
	inboxQueries "github.com/felixgeelhaar/orbita/internal/inbox/application/queries"
	meetingQueries "github.com/felixgeelhaar/orbita/internal/meetings/application/queries"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
)

// SearchResultDTO represents a unified search result.
type SearchResultDTO struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"` // "task", "habit", "meeting", "inbox"
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Status      string         `json:"status,omitempty"`
	Priority    string         `json:"priority,omitempty"`
	DueDate     string         `json:"due_date,omitempty"`
	Score       float64        `json:"score"` // Relevance score
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// SearchResultsDTO represents search results with metadata.
type SearchResultsDTO struct {
	Query      string            `json:"query"`
	TotalCount int               `json:"total_count"`
	Results    []SearchResultDTO `json:"results"`
	Facets     map[string]int    `json:"facets,omitempty"` // Count by type
}

type searchAllInput struct {
	Query    string   `json:"query" jsonschema:"required"`
	Types    []string `json:"types,omitempty"`    // Filter by type: task, habit, meeting, inbox
	Status   string   `json:"status,omitempty"`   // Filter by status
	Priority string   `json:"priority,omitempty"` // Filter by priority
	Limit    int      `json:"limit,omitempty"`    // Max results (default 20)
}

type searchRecentInput struct {
	Types []string `json:"types,omitempty"`
	Limit int      `json:"limit,omitempty"`
}

type searchDueSoonInput struct {
	Days  int `json:"days,omitempty"` // Number of days (default 7)
	Limit int `json:"limit,omitempty"`
}

func registerSearchTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("search.all").
		Description("Search across all items (tasks, habits, meetings, inbox)").
		Handler(func(ctx context.Context, input searchAllInput) (*SearchResultsDTO, error) {
			if app == nil {
				return nil, errors.New("search requires database connection")
			}

			if input.Query == "" {
				return nil, errors.New("query is required")
			}

			query := strings.ToLower(input.Query)
			limit := input.Limit
			if limit <= 0 {
				limit = 20
			}

			var results []SearchResultDTO
			facets := make(map[string]int)

			// Determine which types to search
			searchTypes := map[string]bool{
				"task":    true,
				"habit":   true,
				"meeting": true,
				"inbox":   true,
			}
			if len(input.Types) > 0 {
				searchTypes = make(map[string]bool)
				for _, t := range input.Types {
					searchTypes[t] = true
				}
			}

			// Search tasks
			if searchTypes["task"] && app.ListTasksHandler != nil {
				tasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
					UserID: app.CurrentUserID,
					Status: input.Status,
				})
				if err == nil {
					for _, t := range tasks {
						if matchesQuery(t.Title, query) || matchesQuery(t.Description, query) {
							score := calculateRelevance(t.Title, t.Description, query)
							if input.Priority != "" && t.Priority != input.Priority {
								continue
							}
							var dueDate string
							if t.DueDate != nil {
								dueDate = t.DueDate.Format(dateLayout)
							}
							results = append(results, SearchResultDTO{
								ID:          t.ID.String(),
								Type:        "task",
								Title:       t.Title,
								Description: t.Description,
								Status:      t.Status,
								Priority:    t.Priority,
								DueDate:     dueDate,
								Score:       score,
							})
							facets["task"]++
						}
					}
				}
			}

			// Search habits
			if searchTypes["habit"] && app.ListHabitsHandler != nil {
				habits, err := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
					UserID: app.CurrentUserID,
				})
				if err == nil {
					for _, h := range habits {
						if matchesQuery(h.Name, query) || matchesQuery(h.Description, query) {
							score := calculateRelevance(h.Name, h.Description, query)
							status := "active"
							if h.IsArchived {
								status = "archived"
							}
							results = append(results, SearchResultDTO{
								ID:          h.ID.String(),
								Type:        "habit",
								Title:       h.Name,
								Description: h.Description,
								Status:      status,
								Score:       score,
								Metadata: map[string]any{
									"frequency":      h.Frequency,
									"current_streak": h.Streak,
								},
							})
							facets["habit"]++
						}
					}
				}
			}

			// Search meetings
			if searchTypes["meeting"] && app.ListMeetingsHandler != nil {
				meetings, err := app.ListMeetingsHandler.Handle(ctx, meetingQueries.ListMeetingsQuery{
					UserID: app.CurrentUserID,
				})
				if err == nil {
					for _, m := range meetings {
						if matchesQuery(m.Name, query) {
							score := calculateRelevance(m.Name, "", query)
							status := "active"
							if m.Archived {
								status = "archived"
							}
							results = append(results, SearchResultDTO{
								ID:     m.ID.String(),
								Type:   "meeting",
								Title:  m.Name,
								Status: status,
								Score:  score,
								Metadata: map[string]any{
									"cadence":       m.Cadence,
									"duration_mins": m.DurationMins,
								},
							})
							facets["meeting"]++
						}
					}
				}
			}

			// Search inbox items
			if searchTypes["inbox"] && app.ListInboxItemsHandler != nil {
				items, err := app.ListInboxItemsHandler.Handle(ctx, inboxQueries.ListInboxItemsQuery{
					UserID: app.CurrentUserID,
				})
				if err == nil {
					for _, i := range items {
						if matchesQuery(i.Content, query) {
							score := calculateRelevance(i.Content, "", query)
							status := "pending"
							if i.Promoted {
								status = "promoted"
							}
							results = append(results, SearchResultDTO{
								ID:     i.ID.String(),
								Type:   "inbox",
								Title:  truncate(i.Content, 100),
								Status: status,
								Score:  score,
								Metadata: map[string]any{
									"source":         i.Source,
									"classification": i.Classification,
								},
							})
							facets["inbox"]++
						}
					}
				}
			}

			// Sort by relevance score
			sortByScore(results)

			// Apply limit
			if len(results) > limit {
				results = results[:limit]
			}

			return &SearchResultsDTO{
				Query:      input.Query,
				TotalCount: len(results),
				Results:    results,
				Facets:     facets,
			}, nil
		})

	srv.Tool("search.recent").
		Description("Get recently created or modified items").
		Handler(func(ctx context.Context, input searchRecentInput) (*SearchResultsDTO, error) {
			if app == nil {
				return nil, errors.New("search requires database connection")
			}

			limit := input.Limit
			if limit <= 0 {
				limit = 10
			}

			var results []SearchResultDTO
			facets := make(map[string]int)

			// Get recent tasks
			if app.ListTasksHandler != nil {
				tasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
					UserID:    app.CurrentUserID,
					SortBy:    "created_at",
					SortOrder: "desc",
					Limit:     limit,
				})
				if err == nil {
					for _, t := range tasks {
						var dueDate string
						if t.DueDate != nil {
							dueDate = t.DueDate.Format(dateLayout)
						}
						results = append(results, SearchResultDTO{
							ID:          t.ID.String(),
							Type:        "task",
							Title:       t.Title,
							Description: t.Description,
							Status:      t.Status,
							Priority:    t.Priority,
							DueDate:     dueDate,
							Score:       1.0,
						})
						facets["task"]++
					}
				}
			}

			// Limit total results
			if len(results) > limit {
				results = results[:limit]
			}

			return &SearchResultsDTO{
				Query:      "recent",
				TotalCount: len(results),
				Results:    results,
				Facets:     facets,
			}, nil
		})

	srv.Tool("search.due_soon").
		Description("Find items due within specified days").
		Handler(func(ctx context.Context, input searchDueSoonInput) (*SearchResultsDTO, error) {
			if app == nil || app.ListTasksHandler == nil {
				return nil, errors.New("search requires database connection")
			}

			days := input.Days
			if days <= 0 {
				days = 7
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 20
			}

			dueBefore := time.Now().AddDate(0, 0, days)

			var results []SearchResultDTO
			facets := make(map[string]int)

			// Get tasks due soon
			tasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
				UserID:    app.CurrentUserID,
				DueBefore: &dueBefore,
				Status:    "pending",
				SortBy:    "due_date",
				SortOrder: "asc",
			})
			if err == nil {
				for _, t := range tasks {
					var dueDate string
					if t.DueDate != nil {
						dueDate = t.DueDate.Format(dateLayout)
					}
					results = append(results, SearchResultDTO{
						ID:          t.ID.String(),
						Type:        "task",
						Title:       t.Title,
						Description: t.Description,
						Status:      t.Status,
						Priority:    t.Priority,
						DueDate:     dueDate,
						Score:       1.0,
					})
					facets["task"]++
				}
			}

			if len(results) > limit {
				results = results[:limit]
			}

			return &SearchResultsDTO{
				Query:      "due_soon",
				TotalCount: len(results),
				Results:    results,
				Facets:     facets,
			}, nil
		})

	srv.Tool("search.overdue").
		Description("Find all overdue items").
		Handler(func(ctx context.Context, input struct{}) (*SearchResultsDTO, error) {
			if app == nil || app.ListTasksHandler == nil {
				return nil, errors.New("search requires database connection")
			}

			var results []SearchResultDTO
			facets := make(map[string]int)

			// Get overdue tasks
			tasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
				UserID:  app.CurrentUserID,
				Overdue: true,
			})
			if err == nil {
				for _, t := range tasks {
					var dueDate string
					if t.DueDate != nil {
						dueDate = t.DueDate.Format(dateLayout)
					}
					results = append(results, SearchResultDTO{
						ID:          t.ID.String(),
						Type:        "task",
						Title:       t.Title,
						Description: t.Description,
						Status:      t.Status,
						Priority:    t.Priority,
						DueDate:     dueDate,
						Score:       1.0,
					})
					facets["task"]++
				}
			}

			return &SearchResultsDTO{
				Query:      "overdue",
				TotalCount: len(results),
				Results:    results,
				Facets:     facets,
			}, nil
		})

	return nil
}

func matchesQuery(text, query string) bool {
	return strings.Contains(strings.ToLower(text), query)
}

func calculateRelevance(title, description, query string) float64 {
	score := 0.0
	titleLower := strings.ToLower(title)
	descLower := strings.ToLower(description)

	// Exact match in title = highest score
	if titleLower == query {
		score += 1.0
	} else if strings.HasPrefix(titleLower, query) {
		score += 0.8
	} else if strings.Contains(titleLower, query) {
		score += 0.6
	}

	// Match in description
	if strings.Contains(descLower, query) {
		score += 0.3
	}

	return score
}

func sortByScore(results []SearchResultDTO) {
	// Simple bubble sort for small result sets
	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-i-1; j++ {
			if results[j].Score < results[j+1].Score {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
