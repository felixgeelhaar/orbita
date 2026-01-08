package types

import (
	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/google/uuid"
)

// ClassifierEngine extends the base Engine with classification capabilities.
// Classifier engines analyze text and metadata to categorize inbox items
// into tasks, habits, meetings, or other types.
type ClassifierEngine interface {
	sdk.Engine

	// Classify analyzes content and returns classification results.
	Classify(ctx *sdk.ExecutionContext, input ClassifyInput) (*ClassifyOutput, error)

	// BatchClassify classifies multiple items efficiently.
	BatchClassify(ctx *sdk.ExecutionContext, inputs []ClassifyInput) ([]ClassifyOutput, error)

	// GetCategories returns the categories this engine can classify into.
	GetCategories(ctx *sdk.ExecutionContext) ([]Category, error)
}

// ClassifyInput contains the content to be classified.
type ClassifyInput struct {
	// ID is a unique identifier for this classification request.
	ID uuid.UUID `json:"id"`

	// Content is the text content to classify.
	Content string `json:"content"`

	// Metadata provides additional context for classification.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Source indicates where the content came from.
	Source string `json:"source,omitempty"`

	// Hints are user-provided hints about expected classification.
	Hints []string `json:"hints,omitempty"`
}

// ClassifyOutput contains the classification results.
type ClassifyOutput struct {
	// ID matches the input ID.
	ID uuid.UUID `json:"id"`

	// Category is the primary classification result.
	Category string `json:"category"`

	// Confidence is the confidence level for the primary classification (0-1).
	Confidence float64 `json:"confidence"`

	// Alternatives are other possible classifications with lower confidence.
	Alternatives []ClassificationAlternative `json:"alternatives,omitempty"`

	// ExtractedEntities contains entities extracted from the content.
	ExtractedEntities ExtractedEntities `json:"extracted_entities,omitempty"`

	// Explanation describes why this classification was chosen.
	Explanation string `json:"explanation,omitempty"`

	// RequiresReview indicates if human review is recommended.
	RequiresReview bool `json:"requires_review"`

	// ReviewReason explains why review is recommended.
	ReviewReason string `json:"review_reason,omitempty"`
}

// ClassificationAlternative represents an alternative classification.
type ClassificationAlternative struct {
	// Category is the alternative classification.
	Category string `json:"category"`

	// Confidence is the confidence level for this alternative (0-1).
	Confidence float64 `json:"confidence"`

	// Reason explains why this could be an alternative.
	Reason string `json:"reason,omitempty"`
}

// ExtractedEntities contains structured data extracted from content.
type ExtractedEntities struct {
	// Title is the extracted title or summary.
	Title string `json:"title,omitempty"`

	// Description is the extracted description.
	Description string `json:"description,omitempty"`

	// DueDate is an extracted deadline.
	DueDate string `json:"due_date,omitempty"`

	// Duration is an extracted time estimate.
	Duration string `json:"duration,omitempty"`

	// Priority is an extracted priority level.
	Priority string `json:"priority,omitempty"`

	// People are names of people mentioned.
	People []string `json:"people,omitempty"`

	// Tags are extracted tags or categories.
	Tags []string `json:"tags,omitempty"`

	// URLs are extracted URLs.
	URLs []string `json:"urls,omitempty"`

	// Custom contains engine-specific extracted data.
	Custom map[string]any `json:"custom,omitempty"`
}

// Category represents a classification category.
type Category struct {
	// ID is the category identifier.
	ID string `json:"id"`

	// Name is the human-readable name.
	Name string `json:"name"`

	// Description explains what this category represents.
	Description string `json:"description"`

	// Examples are example inputs for this category.
	Examples []string `json:"examples,omitempty"`

	// Keywords are trigger words for this category.
	Keywords []string `json:"keywords,omitempty"`
}

// StandardCategories defines the standard Orbita categories.
var StandardCategories = []Category{
	{
		ID:          "task",
		Name:        "Task",
		Description: "A one-time actionable item with a clear completion criteria",
		Examples:    []string{"Review PR #123", "Submit expense report", "Call dentist for appointment"},
		Keywords:    []string{"do", "complete", "finish", "submit", "review", "send", "create"},
	},
	{
		ID:          "habit",
		Name:        "Habit",
		Description: "A recurring activity to build or maintain a behavior",
		Examples:    []string{"Exercise for 30 minutes", "Read 10 pages", "Meditate daily"},
		Keywords:    []string{"daily", "weekly", "habit", "routine", "every day", "each morning"},
	},
	{
		ID:          "meeting",
		Name:        "Meeting",
		Description: "A scheduled interaction with one or more people",
		Examples:    []string{"1:1 with John", "Team standup", "Client call with Acme"},
		Keywords:    []string{"meeting", "call", "1:1", "sync", "chat with", "discuss with"},
	},
	{
		ID:          "note",
		Name:        "Note",
		Description: "Information to capture without immediate action required",
		Examples:    []string{"Ideas for Q2 planning", "Book recommendation from Sarah"},
		Keywords:    []string{"remember", "note", "idea", "thought", "save"},
	},
	{
		ID:          "event",
		Name:        "Event",
		Description: "A time-bound occurrence that doesn't require direct action",
		Examples:    []string{"Company all-hands", "Conference", "Holiday"},
		Keywords:    []string{"event", "conference", "holiday", "birthday", "anniversary"},
	},
}

// ClassifierEngineCapabilities defines what a classifier engine can do.
const (
	// CapabilityClassify indicates basic classification.
	CapabilityClassify = "classify"

	// CapabilityBatchClassify indicates batch processing support.
	CapabilityBatchClassify = "batch_classify"

	// CapabilityEntityExtraction indicates entity extraction.
	CapabilityEntityExtraction = "entity_extraction"

	// CapabilityNLU indicates natural language understanding.
	CapabilityNLU = "nlu"

	// CapabilitySentimentAnalysis indicates sentiment analysis.
	CapabilitySentimentAnalysis = "sentiment_analysis"

	// CapabilityMultiLabel indicates support for multiple labels.
	CapabilityMultiLabel = "multi_label"
)
