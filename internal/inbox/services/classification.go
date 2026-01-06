package services

import (
	"strings"

	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
)

// Classifier determines a suggested classification from content/metadata.
type Classifier struct{}

// NewClassifier returns a classifier instance.
func NewClassifier() *Classifier {
	return &Classifier{}
}

// Classify returns classification label.
func (c *Classifier) Classify(content string, metadata domain.InboxMetadata) string {
	text := strings.ToLower(content)
	if metadataType, ok := metadata["type"]; ok {
		switch metadataType {
		case "task", "habit", "meeting":
			return metadataType
		}
	}
	if strings.Contains(text, "meeting") || strings.Contains(text, "call") {
		return "meeting"
	}
	if strings.Contains(text, "habit") || strings.Contains(text, "daily") {
		return "habit"
	}
	return "task"
}
