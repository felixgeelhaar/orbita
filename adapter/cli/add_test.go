package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExtractPriority(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedPrio   string
		expectedOutput string
	}{
		{
			name:           "triple exclamation marks",
			input:          "Fix bug !!!",
			expectedPrio:   "urgent",
			expectedOutput: "Fix bug ",
		},
		{
			name:           "double exclamation marks",
			input:          "Review PR !!",
			expectedPrio:   "high",
			expectedOutput: "Review PR ",
		},
		{
			name:           "single exclamation mark",
			input:          "Call mom !",
			expectedPrio:   "medium",
			expectedOutput: "Call mom ",
		},
		{
			name:           "urgent keyword alone",
			input:          "Fix critical bug urgent",
			expectedPrio:   "urgent",
			expectedOutput: "Fix critical bug ",
		},
		{
			name:           "high keyword alone",
			input:          "Complete report high",
			expectedPrio:   "high",
			expectedOutput: "Complete report ",
		},
		{
			name:           "low keyword alone",
			input:          "Organize files low",
			expectedPrio:   "low",
			expectedOutput: "Organize files ",
		},
		{
			name:           "no priority",
			input:          "Buy groceries",
			expectedPrio:   "",
			expectedOutput: "Buy groceries",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			priority, output := extractPriority(tc.input)
			assert.Equal(t, tc.expectedPrio, priority)
			assert.Equal(t, tc.expectedOutput, output)
		})
	}
}

func TestExtractDuration(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedDuration time.Duration
		expectedOutput   string
	}{
		{
			name:             "30min",
			input:            "Review code 30min",
			expectedDuration: 30 * time.Minute,
			expectedOutput:   "Review code ",
		},
		{
			name:             "1h",
			input:            "Deep work 1h",
			expectedDuration: 1 * time.Hour,
			expectedOutput:   "Deep work ",
		},
		{
			name:             "2 hours",
			input:            "Meeting 2 hours",
			expectedDuration: 2 * time.Hour,
			expectedOutput:   "Meeting ",
		},
		{
			name:             "1.5h",
			input:            "Workshop 1.5h",
			expectedDuration: 90 * time.Minute,
			expectedOutput:   "Workshop ",
		},
		{
			name:             "for 30 minutes",
			input:            "Break for 30 minutes",
			expectedDuration: 30 * time.Minute,
			expectedOutput:   "Break for ", // only the duration part is removed
		},
		{
			name:             "for 2 hours",
			input:            "Study for 2 hours",
			expectedDuration: 2 * time.Hour,
			expectedOutput:   "Study for ", // only the duration part is removed
		},
		{
			name:             "no duration",
			input:            "Buy groceries",
			expectedDuration: 0,
			expectedOutput:   "Buy groceries",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			duration, output := extractDuration(tc.input)
			assert.Equal(t, tc.expectedDuration, duration)
			assert.Equal(t, tc.expectedOutput, output)
		})
	}
}

func TestExtractDueDate(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	tests := []struct {
		name           string
		input          string
		expectedDate   *time.Time
		expectedOutput string
	}{
		{
			name:           "today",
			input:          "Buy groceries today",
			expectedDate:   &today,
			expectedOutput: "Buy groceries ",
		},
		{
			name:           "tomorrow",
			input:          "Call mom tomorrow",
			expectedDate:   func() *time.Time { d := today.AddDate(0, 0, 1); return &d }(),
			expectedOutput: "Call mom ",
		},
		{
			name:           "next week",
			input:          "Review report next week",
			expectedDate:   func() *time.Time { d := today.AddDate(0, 0, 7); return &d }(),
			expectedOutput: "Review report ",
		},
		{
			name:           "specific date YYYY-MM-DD",
			input:          "Submit proposal 2026-06-15",
			expectedDate:   func() *time.Time { d := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC); return &d }(),
			expectedOutput: "Submit proposal ",
		},
		{
			name:           "no due date",
			input:          "Buy groceries",
			expectedDate:   nil,
			expectedOutput: "Buy groceries",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			date, output := extractDueDate(tc.input)
			if tc.expectedDate == nil {
				assert.Nil(t, date)
			} else {
				assert.NotNil(t, date)
				assert.Equal(t, tc.expectedDate.Year(), date.Year())
				assert.Equal(t, tc.expectedDate.Month(), date.Month())
				assert.Equal(t, tc.expectedDate.Day(), date.Day())
			}
			assert.Equal(t, tc.expectedOutput, output)
		})
	}
}

func TestExtractDueDate_Weekdays(t *testing.T) {
	// Test weekday parsing - dates will vary based on current day
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	weekdays := []struct {
		input   string
		weekday time.Weekday
	}{
		{"Meeting monday", time.Monday},
		{"Meeting tuesday", time.Tuesday},
		{"Meeting wednesday", time.Wednesday},
		{"Meeting thursday", time.Thursday},
		{"Meeting friday", time.Friday},
		{"Meeting saturday", time.Saturday},
		{"Meeting sunday", time.Sunday},
	}

	for _, tc := range weekdays {
		t.Run(tc.input, func(t *testing.T) {
			date, _ := extractDueDate(tc.input)
			assert.NotNil(t, date)

			// Calculate expected date
			expected := nextWeekday(today, tc.weekday)
			assert.Equal(t, expected.Year(), date.Year())
			assert.Equal(t, expected.Month(), date.Month())
			assert.Equal(t, expected.Day(), date.Day())
		})
	}
}

func TestExtractDueDate_ByWeekday(t *testing.T) {
	// Test "by monday" format
	date, output := extractDueDate("Finish report by friday")
	assert.NotNil(t, date)
	assert.Equal(t, time.Friday, date.Weekday())
	assert.Contains(t, output, "Finish report")
}

func TestExtractDueDate_NextWeekday(t *testing.T) {
	// Test "next monday" format
	date, output := extractDueDate("Submit proposal next monday")
	assert.NotNil(t, date)
	assert.Equal(t, time.Monday, date.Weekday())
	assert.Contains(t, output, "Submit proposal")
}

func TestNextWeekday(t *testing.T) {
	// Test from a known date (Wednesday)
	wednesday := time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC) // Jan 7, 2026 is a Wednesday

	tests := []struct {
		target   time.Weekday
		expected time.Time
	}{
		{time.Monday, time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC)},    // Next Monday
		{time.Tuesday, time.Date(2026, 1, 13, 0, 0, 0, 0, time.UTC)},   // Next Tuesday
		{time.Wednesday, time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC)}, // Next Wednesday (not same day)
		{time.Thursday, time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC)},   // Tomorrow
		{time.Friday, time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)},     // Day after tomorrow
		{time.Saturday, time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)},  // 3 days
		{time.Sunday, time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC)},    // 4 days
	}

	for _, tc := range tests {
		t.Run(tc.target.String(), func(t *testing.T) {
			result := nextWeekday(wednesday, tc.target)
			assert.Equal(t, tc.expected.Year(), result.Year())
			assert.Equal(t, tc.expected.Month(), result.Month())
			assert.Equal(t, tc.expected.Day(), result.Day())
		})
	}
}

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "extra spaces",
			input:    "Buy   groceries   today",
			expected: "Buy groceries today",
		},
		{
			name:     "leading filler word",
			input:    "by Friday complete report",
			expected: "Friday complete report",
		},
		{
			name:     "trailing filler word",
			input:    "Complete report by",
			expected: "Complete report",
		},
		{
			name:     "leading 'for'",
			input:    "for the team meeting notes",
			expected: "the team meeting notes",
		},
		{
			name:     "trailing 'for'",
			input:    "meeting notes for",
			expected: "meeting notes",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  Buy groceries  ",
			expected: "Buy groceries",
		},
		{
			name:     "normal input",
			input:    "Buy groceries",
			expected: "Buy groceries",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := cleanTitle(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseNaturalLanguage(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedTitle string
		expectedPrio  string
		hasDuration   bool
		hasDueDate    bool
	}{
		{
			name:          "simple task",
			input:         "Buy groceries",
			expectedTitle: "Buy groceries",
			expectedPrio:  "",
			hasDuration:   false,
			hasDueDate:    false,
		},
		{
			name:          "task with priority",
			input:         "Fix critical bug urgent",
			expectedTitle: "Fix critical bug",
			expectedPrio:  "urgent",
			hasDuration:   false,
			hasDueDate:    false,
		},
		{
			name:          "task with duration",
			input:         "Code review 30min",
			expectedTitle: "Code review",
			expectedPrio:  "",
			hasDuration:   true,
			hasDueDate:    false,
		},
		{
			name:          "task with due date",
			input:         "Submit report tomorrow",
			expectedTitle: "Submit report",
			expectedPrio:  "",
			hasDuration:   false,
			hasDueDate:    true,
		},
		{
			name:          "complex task",
			input:         "Team meeting monday 2h urgent",
			expectedTitle: "Team meeting",
			expectedPrio:  "urgent",
			hasDuration:   true,
			hasDueDate:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseNaturalLanguage(tc.input)
			assert.Equal(t, tc.expectedTitle, result.title)
			assert.Equal(t, tc.expectedPrio, result.priority)

			if tc.hasDuration {
				assert.Greater(t, result.duration, time.Duration(0))
			} else {
				assert.Equal(t, time.Duration(0), result.duration)
			}

			if tc.hasDueDate {
				assert.NotNil(t, result.dueDate)
			} else {
				assert.Nil(t, result.dueDate)
			}
		})
	}
}
