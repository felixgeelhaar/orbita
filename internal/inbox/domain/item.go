package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// InboxMetadata is user-supplied metadata stored with the item.
type InboxMetadata map[string]string

// InboxItem represents a captured idea or request.
type InboxItem struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Content        string
	Metadata       InboxMetadata
	Tags           []string
	Source         string
	Classification string
	CapturedAt     time.Time
	Promoted       bool
	PromotedTo     string
	PromotedID     uuid.UUID
	PromotedAt     *time.Time
}

// MarshalJSON ensures metadata is encoded as JSON.
func (m InboxMetadata) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string(m))
}

// UnmarshalJSON decodes metadata JSON.
func (m *InboxMetadata) UnmarshalJSON(b []byte) error {
	values := map[string]string{}
	if err := json.Unmarshal(b, &values); err != nil {
		return err
	}
	*m = values
	return nil
}
