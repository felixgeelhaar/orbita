package cli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <description>",
	Short: "Quick add a task with natural language",
	Long: `Quickly add a task using natural language.

The command parses your input to extract:
- Task title (required)
- Due date: today, tomorrow, next week, monday-sunday, or YYYY-MM-DD
- Priority: urgent, high, medium, low (or !, !!, !!!)
- Duration: 30min, 1h, 2 hours, etc.

Examples:
  orbita add "Buy groceries"
  orbita add "Buy groceries tomorrow"
  orbita add "Finish report by friday high priority"
  orbita add "Call mom today !!"
  orbita add "Review PR for 30min"
  orbita add "Team meeting next monday 2h urgent"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.CreateTaskHandler == nil {
			fmt.Println("Quick add requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		input := strings.Join(args, " ")
		parsed := parseNaturalLanguage(input)

		// Build command
		durationMins := 0
		if parsed.duration > 0 {
			durationMins = int(parsed.duration.Minutes())
		}

		createCmd := commands.CreateTaskCommand{
			UserID:          app.CurrentUserID,
			Title:           parsed.title,
			Description:     "",
			Priority:        parsed.priority,
			DurationMinutes: durationMins,
			DueDate:         parsed.dueDate,
		}

		result, err := app.CreateTaskHandler.Handle(cmd.Context(), createCmd)
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		fmt.Println("Task created!")
		fmt.Printf("  Title: %s\n", parsed.title)
		fmt.Printf("  ID: %s\n", result.TaskID.String()[:8])

		if parsed.priority != "" {
			fmt.Printf("  Priority: %s\n", parsed.priority)
		}
		if parsed.duration > 0 {
			fmt.Printf("  Duration: %d min\n", int(parsed.duration.Minutes()))
		}
		if parsed.dueDate != nil {
			fmt.Printf("  Due: %s\n", parsed.dueDate.Format("Mon, Jan 2 2006"))
		}

		return nil
	},
}

type parsedInput struct {
	title    string
	priority string
	duration time.Duration
	dueDate  *time.Time
}

func parseNaturalLanguage(input string) parsedInput {
	result := parsedInput{
		title: input,
	}

	// Extract priority
	result.priority, result.title = extractPriority(result.title)

	// Extract duration
	result.duration, result.title = extractDuration(result.title)

	// Extract due date
	result.dueDate, result.title = extractDueDate(result.title)

	// Clean up title
	result.title = cleanTitle(result.title)

	return result
}

func extractPriority(input string) (string, string) {
	// Check for !!! or !! or !
	if strings.Contains(input, "!!!") {
		return "urgent", strings.ReplaceAll(input, "!!!", "")
	}
	if strings.Contains(input, "!!") {
		return "high", strings.ReplaceAll(input, "!!", "")
	}
	if strings.Contains(input, "!") && !strings.Contains(input, "!!") {
		return "medium", strings.ReplaceAll(input, "!", "")
	}

	// Check for priority keywords
	lower := strings.ToLower(input)
	priorities := map[string]string{
		"urgent priority": "urgent",
		"high priority":   "high",
		"medium priority": "medium",
		"low priority":    "low",
		"urgent":          "urgent",
		"high":            "high",
		"low":             "low",
	}

	for keyword, priority := range priorities {
		if strings.Contains(lower, keyword) {
			// Remove the keyword from input
			re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(keyword) + `\b`)
			return priority, re.ReplaceAllString(input, "")
		}
	}

	return "", input
}

func extractDuration(input string) (time.Duration, string) {
	// Patterns: 30min, 30 min, 1h, 1 hour, 2 hours, 1.5h
	patterns := []struct {
		regex      *regexp.Regexp
		multiplier time.Duration
	}{
		{regexp.MustCompile(`(\d+(?:\.\d+)?)\s*h(?:ours?)?`), time.Hour},
		{regexp.MustCompile(`(\d+)\s*min(?:utes?)?`), time.Minute},
		{regexp.MustCompile(`for\s+(\d+(?:\.\d+)?)\s*h(?:ours?)?`), time.Hour},
		{regexp.MustCompile(`for\s+(\d+)\s*min(?:utes?)?`), time.Minute},
	}

	for _, p := range patterns {
		if matches := p.regex.FindStringSubmatch(strings.ToLower(input)); len(matches) > 1 {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				duration := time.Duration(val * float64(p.multiplier))
				cleaned := p.regex.ReplaceAllString(input, "")
				return duration, cleaned
			}
		}
	}

	return 0, input
}

func extractDueDate(input string) (*time.Time, string) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lower := strings.ToLower(input)

	// Check for relative dates
	relativeDates := map[string]time.Time{
		"today":     today,
		"tomorrow":  today.AddDate(0, 0, 1),
		"next week": today.AddDate(0, 0, 7),
	}

	for keyword, date := range relativeDates {
		if strings.Contains(lower, keyword) {
			d := date
			re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(keyword) + `\b`)
			return &d, re.ReplaceAllString(input, "")
		}
	}

	// Check for "by <day>" or just day names
	days := map[string]time.Weekday{
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
		"sunday":    time.Sunday,
	}

	for dayName, weekday := range days {
		// Check for "by monday", "next monday", or just "monday"
		patterns := []string{
			`by\s+` + dayName,
			`next\s+` + dayName,
			`\b` + dayName + `\b`,
		}

		for _, pattern := range patterns {
			re := regexp.MustCompile(`(?i)` + pattern)
			if re.MatchString(lower) {
				date := nextWeekday(today, weekday)
				return &date, re.ReplaceAllString(input, "")
			}
		}
	}

	// Check for YYYY-MM-DD format
	datePattern := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	if matches := datePattern.FindStringSubmatch(input); len(matches) > 1 {
		if date, err := time.Parse("2006-01-02", matches[1]); err == nil {
			return &date, datePattern.ReplaceAllString(input, "")
		}
	}

	return nil, input
}

func nextWeekday(from time.Time, target time.Weekday) time.Time {
	daysUntil := int(target) - int(from.Weekday())
	if daysUntil <= 0 {
		daysUntil += 7
	}
	return from.AddDate(0, 0, daysUntil)
}

func cleanTitle(title string) string {
	// Remove extra whitespace
	re := regexp.MustCompile(`\s+`)
	title = re.ReplaceAllString(title, " ")

	// Remove common filler words at boundaries
	fillers := []string{"by", "for", "at", "on"}
	for _, filler := range fillers {
		// Remove if at start or end
		title = regexp.MustCompile(`(?i)^\s*` + filler + `\s+`).ReplaceAllString(title, "")
		title = regexp.MustCompile(`(?i)\s+` + filler + `\s*$`).ReplaceAllString(title, "")
	}

	return strings.TrimSpace(title)
}

func init() {
	rootCmd.AddCommand(addCmd)
}
