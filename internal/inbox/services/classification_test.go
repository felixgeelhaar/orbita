package services

import (
	"testing"

	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewClassifier(t *testing.T) {
	c := NewClassifier()
	assert.NotNil(t, c)
}

func TestClassifier_Classify(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name     string
		content  string
		metadata domain.InboxMetadata
		expected string
	}{
		{
			name:     "metadata type task",
			content:  "some content",
			metadata: domain.InboxMetadata{"type": "task"},
			expected: "task",
		},
		{
			name:     "metadata type habit",
			content:  "some content",
			metadata: domain.InboxMetadata{"type": "habit"},
			expected: "habit",
		},
		{
			name:     "metadata type meeting",
			content:  "some content",
			metadata: domain.InboxMetadata{"type": "meeting"},
			expected: "meeting",
		},
		{
			name:     "content contains meeting",
			content:  "Schedule a meeting with John",
			metadata: domain.InboxMetadata{},
			expected: "meeting",
		},
		{
			name:     "content contains call",
			content:  "Call the team tomorrow",
			metadata: domain.InboxMetadata{},
			expected: "meeting",
		},
		{
			name:     "content contains habit",
			content:  "Start a new habit of reading",
			metadata: domain.InboxMetadata{},
			expected: "habit",
		},
		{
			name:     "content contains daily",
			content:  "Daily standup reminder",
			metadata: domain.InboxMetadata{},
			expected: "habit",
		},
		{
			name:     "default to task",
			content:  "Fix the bug in login page",
			metadata: domain.InboxMetadata{},
			expected: "task",
		},
		{
			name:     "empty content defaults to task",
			content:  "",
			metadata: domain.InboxMetadata{},
			expected: "task",
		},
		{
			name:     "case insensitive matching",
			content:  "MEETING with boss",
			metadata: domain.InboxMetadata{},
			expected: "meeting",
		},
		{
			name:     "metadata takes precedence over content",
			content:  "meeting content",
			metadata: domain.InboxMetadata{"type": "task"},
			expected: "task",
		},
		{
			name:     "unknown metadata type falls through to content analysis",
			content:  "meeting tomorrow",
			metadata: domain.InboxMetadata{"type": "unknown"},
			expected: "meeting",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := c.Classify(tc.content, tc.metadata)
			assert.Equal(t, tc.expected, result)
		})
	}
}
