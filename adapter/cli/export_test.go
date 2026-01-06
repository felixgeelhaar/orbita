package cli

import (
	"strings"
	"testing"
	"time"

	scheduleQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	"github.com/google/uuid"
)

func TestFormatICSTime(t *testing.T) {
	ts := time.Date(2024, time.May, 1, 13, 14, 15, 0, time.FixedZone("UTC+2", 2*60*60))
	formatted := formatICSTime(ts)
	if formatted != "20240501T111415Z" {
		t.Fatalf("unexpected format: %s", formatted)
	}
}

func TestEscapeICS(t *testing.T) {
	input := "One\\Two;Three,Four\nFive"
	escaped := escapeICS(input)
	expected := "One\\\\Two\\;Three\\,Four\\nFive"
	if escaped != expected {
		t.Fatalf("unexpected escape result: %s", escaped)
	}
}

func TestGenerateICS_SingleBlock(t *testing.T) {
	blockID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	start := time.Date(2024, time.May, 2, 9, 0, 0, 0, time.UTC)
	end := time.Date(2024, time.May, 2, 9, 30, 0, 0, time.UTC)

	blocks := []scheduleQueries.TimeBlockDTO{
		{
			ID:        blockID,
			BlockType: "task",
			Title:     "Standup, team; sync",
			StartTime: start,
			EndTime:   end,
			Completed: true,
		},
	}

	ics := generateICS(blocks)

	assertContains(t, ics, "BEGIN:VCALENDAR\r\n")
	assertContains(t, ics, "BEGIN:VEVENT\r\n")
	assertContains(t, ics, "UID:11111111-1111-1111-1111-111111111111@orbita\r\n")
	assertContains(t, ics, "DTSTART:20240502T090000Z\r\n")
	assertContains(t, ics, "DTEND:20240502T093000Z\r\n")
	assertContains(t, ics, "SUMMARY:Standup\\, team\\; sync\r\n")
	assertContains(t, ics, "DESCRIPTION:Type: task\\nStatus: Completed\r\n")
	assertContains(t, ics, "CATEGORIES:TASK\r\n")
	assertContains(t, ics, "STATUS:CONFIRMED\r\n")
	assertContains(t, ics, "END:VEVENT\r\n")
	assertContains(t, ics, "END:VCALENDAR\r\n")
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected to contain %q", needle)
	}
}
