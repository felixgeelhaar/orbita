package builtin

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
)

// ClassifierEnginePro is an advanced classifier with NLU capabilities,
// intelligent entity extraction, and context-aware classification.
type ClassifierEnginePro struct {
	config     sdk.EngineConfig
	categories []types.Category
}

// NewClassifierEnginePro creates a new pro classifier engine.
func NewClassifierEnginePro() *ClassifierEnginePro {
	return &ClassifierEnginePro{
		categories: types.StandardCategories,
	}
}

// Metadata returns engine metadata.
func (e *ClassifierEnginePro) Metadata() sdk.EngineMetadata {
	return sdk.EngineMetadata{
		ID:            "orbita.classifier.pro",
		Name:          "AI Inbox Pro",
		Version:       "1.0.0",
		Author:        "Orbita",
		Description:   "Advanced AI-powered inbox classification with NLU, entity extraction, and intelligent categorization",
		License:       "Proprietary",
		Homepage:      "https://orbita.app",
		Tags:          []string{"classifier", "pro", "ai-inbox", "nlu", "entity-extraction"},
		MinAPIVersion: "1.0.0",
		Capabilities: []string{
			types.CapabilityClassify,
			types.CapabilityBatchClassify,
			types.CapabilityEntityExtraction,
			types.CapabilityNLU,
			types.CapabilitySentimentAnalysis,
			types.CapabilityMultiLabel,
		},
	}
}

// Type returns the engine type.
func (e *ClassifierEnginePro) Type() sdk.EngineType {
	return sdk.EngineTypeClassifier
}

// ConfigSchema returns the configuration schema.
func (e *ClassifierEnginePro) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		Properties: map[string]sdk.PropertySchema{
			// Classification Settings
			"confidence_threshold": {
				Type:        "number",
				Title:       "Confidence Threshold",
				Description: "Minimum confidence level for classification (0-1)",
				Default:     0.7,
				Minimum:     floatPtr(0.1),
				Maximum:     floatPtr(0.99),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Classification",
					Order:  1,
				},
			},
			"multi_label_enabled": {
				Type:        "boolean",
				Title:       "Enable Multi-Label",
				Description: "Allow items to be classified into multiple categories",
				Default:     false,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Classification",
					Order:  2,
				},
			},
			"review_low_confidence": {
				Type:        "boolean",
				Title:       "Flag Low Confidence",
				Description: "Flag items with low confidence for human review",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Classification",
					Order:  3,
				},
			},
			"review_threshold": {
				Type:        "number",
				Title:       "Review Threshold",
				Description: "Items below this confidence are flagged for review",
				Default:     0.5,
				Minimum:     floatPtr(0.1),
				Maximum:     floatPtr(0.9),
				UIHints: sdk.UIHints{
					Widget: "slider",
					Group:  "Classification",
					Order:  4,
				},
			},

			// Entity Extraction Settings
			"extract_dates": {
				Type:        "boolean",
				Title:       "Extract Dates",
				Description: "Extract and parse dates from content",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Entity Extraction",
					Order:  1,
				},
			},
			"extract_durations": {
				Type:        "boolean",
				Title:       "Extract Durations",
				Description: "Extract time durations from content",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Entity Extraction",
					Order:  2,
				},
			},
			"extract_people": {
				Type:        "boolean",
				Title:       "Extract People",
				Description: "Extract names of people from content",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Entity Extraction",
					Order:  3,
				},
			},
			"extract_priorities": {
				Type:        "boolean",
				Title:       "Extract Priorities",
				Description: "Detect priority levels from content",
				Default:     true,
				UIHints: sdk.UIHints{
					Widget: "toggle",
					Group:  "Entity Extraction",
					Order:  4,
				},
			},

			// Custom Categories
			"custom_keywords": {
				Type:        "object",
				Title:       "Custom Keywords",
				Description: "Map of category to additional keywords",
				Default:     map[string]any{},
				UIHints: sdk.UIHints{
					Widget: "json",
					Group:  "Customization",
					Order:  1,
				},
			},
		},
		Required: []string{},
	}
}

// Initialize initializes the engine with configuration.
func (e *ClassifierEnginePro) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	e.config = config
	return nil
}

// HealthCheck returns the engine health status.
func (e *ClassifierEnginePro) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{
		Healthy: true,
		Message: "AI Inbox Pro is healthy",
	}
}

// Shutdown gracefully shuts down the engine.
func (e *ClassifierEnginePro) Shutdown(ctx context.Context) error {
	return nil
}

// Classify analyzes content and returns classification results.
func (e *ClassifierEnginePro) Classify(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error) {
	ctx.Logger.Debug("classifying content",
		"id", input.ID,
		"content_length", len(input.Content),
		"source", input.Source,
	)

	// Score each category
	scores := e.scoreCategories(input.Content, input.Hints)

	// Find best match
	var bestCategory string
	var bestScore float64
	for category, score := range scores {
		if score > bestScore {
			bestScore = score
			bestCategory = category
		}
	}

	// Build alternatives
	alternatives := e.buildAlternatives(scores, bestCategory)

	// Extract entities
	entities := e.extractEntities(input.Content)

	// Determine if review is needed
	reviewThreshold := e.getFloat("review_threshold", 0.5)
	requiresReview := e.getBool("review_low_confidence", true) && bestScore < reviewThreshold
	reviewReason := ""
	if requiresReview {
		reviewReason = e.determineReviewReason(bestScore, alternatives, input.Content)
	}

	return &types.ClassifyOutput{
		ID:                input.ID,
		Category:          bestCategory,
		Confidence:        bestScore,
		Alternatives:      alternatives,
		ExtractedEntities: entities,
		Explanation:       e.generateExplanation(bestCategory, bestScore, input.Content),
		RequiresReview:    requiresReview,
		ReviewReason:      reviewReason,
	}, nil
}

// BatchClassify classifies multiple items efficiently.
func (e *ClassifierEnginePro) BatchClassify(ctx *sdk.ExecutionContext, inputs []types.ClassifyInput) ([]types.ClassifyOutput, error) {
	ctx.Logger.Debug("batch classifying", "count", len(inputs))

	outputs := make([]types.ClassifyOutput, len(inputs))
	for i, input := range inputs {
		output, err := e.Classify(ctx, input)
		if err != nil {
			// Add failed result
			outputs[i] = types.ClassifyOutput{
				ID:             input.ID,
				Category:       "unknown",
				Confidence:     0,
				RequiresReview: true,
				ReviewReason:   "classification error: " + err.Error(),
			}
			continue
		}
		outputs[i] = *output
	}

	return outputs, nil
}

// GetCategories returns the available categories.
func (e *ClassifierEnginePro) GetCategories(ctx *sdk.ExecutionContext) ([]types.Category, error) {
	return e.categories, nil
}

// scoreCategories scores content against each category.
func (e *ClassifierEnginePro) scoreCategories(content string, hints []string) map[string]float64 {
	scores := make(map[string]float64)
	contentLower := strings.ToLower(content)
	words := strings.Fields(contentLower)

	for _, category := range e.categories {
		score := 0.0

		// Keyword matching
		keywordMatches := 0
		for _, keyword := range category.Keywords {
			if strings.Contains(contentLower, strings.ToLower(keyword)) {
				keywordMatches++
			}
		}
		if len(category.Keywords) > 0 {
			score += float64(keywordMatches) / float64(len(category.Keywords)) * 0.4
		}

		// Pattern-based scoring
		score += e.scoreByPatterns(contentLower, category.ID)

		// Apply hints
		for _, hint := range hints {
			if strings.EqualFold(hint, category.ID) || strings.EqualFold(hint, category.Name) {
				score += 0.3 // Significant boost for explicit hints
			}
		}

		// Semantic indicators
		score += e.scoreBySemantics(words, category.ID)

		// Normalize to 0-1 range
		if score > 1.0 {
			score = 1.0
		}
		if score < 0 {
			score = 0
		}

		scores[category.ID] = score
	}

	return scores
}

// scoreByPatterns scores based on linguistic patterns.
func (e *ClassifierEnginePro) scoreByPatterns(content, categoryID string) float64 {
	score := 0.0

	switch categoryID {
	case "task":
		// Imperative verbs at start suggest task
		imperativePatterns := []string{
			`^(do|complete|finish|submit|review|send|create|update|fix|write|make|prepare|check)\b`,
			`need to\b`,
			`must\b`,
			`should\b`,
			`todo\b`,
			`action required`,
		}
		for _, pattern := range imperativePatterns {
			if matched, _ := regexp.MatchString(pattern, content); matched {
				score += 0.15
			}
		}

	case "habit":
		// Recurring patterns
		habitPatterns := []string{
			`every (day|week|morning|evening|monday|tuesday|wednesday|thursday|friday|saturday|sunday)`,
			`daily`,
			`weekly`,
			`regularly`,
			`(build|start|maintain) (a |the )?habit`,
			`routine`,
			`\d+ (minutes?|hours?) (of|for)`,
		}
		for _, pattern := range habitPatterns {
			if matched, _ := regexp.MatchString(pattern, content); matched {
				score += 0.2
			}
		}

	case "meeting":
		// Meeting indicators
		meetingPatterns := []string{
			`meet(ing)? with`,
			`call with`,
			`1(:|-)1|one[- ]on[- ]one`,
			`sync with`,
			`discuss with`,
			`@\s*\d{1,2}(:\d{2})?\s*(am|pm)?`,
			`(monday|tuesday|wednesday|thursday|friday) at`,
		}
		for _, pattern := range meetingPatterns {
			if matched, _ := regexp.MatchString(pattern, content); matched {
				score += 0.2
			}
		}

	case "note":
		// Note indicators
		notePatterns := []string{
			`^(idea|thought|note|remember)`,
			`for (later|reference)`,
			`don't forget`,
			`keep in mind`,
			`fyi`,
			`^note:`,
		}
		for _, pattern := range notePatterns {
			if matched, _ := regexp.MatchString(pattern, content); matched {
				score += 0.2
			}
		}

	case "event":
		// Event indicators
		eventPatterns := []string{
			`(company|team) (event|all-hands|offsite)`,
			`conference`,
			`(birthday|anniversary|holiday)`,
			`happening (on|at)`,
		}
		for _, pattern := range eventPatterns {
			if matched, _ := regexp.MatchString(pattern, content); matched {
				score += 0.2
			}
		}
	}

	return score
}

// scoreBySemantics scores based on semantic analysis.
func (e *ClassifierEnginePro) scoreBySemantics(words []string, categoryID string) float64 {
	score := 0.0

	// Detect action verbs
	actionVerbs := map[string]bool{
		"do": true, "complete": true, "finish": true, "send": true, "create": true,
		"update": true, "fix": true, "review": true, "submit": true, "write": true,
	}

	// Detect time-related words
	timeWords := map[string]bool{
		"daily": true, "weekly": true, "monthly": true, "every": true, "routine": true,
	}

	// Detect meeting words
	meetingWords := map[string]bool{
		"meeting": true, "call": true, "sync": true, "discuss": true, "chat": true, "1:1": true,
	}

	for _, word := range words {
		switch categoryID {
		case "task":
			if actionVerbs[word] {
				score += 0.05
			}
		case "habit":
			if timeWords[word] {
				score += 0.1
			}
		case "meeting":
			if meetingWords[word] {
				score += 0.1
			}
		}
	}

	return score
}

// buildAlternatives creates alternative classifications.
func (e *ClassifierEnginePro) buildAlternatives(scores map[string]float64, primary string) []types.ClassificationAlternative {
	// Sort by score
	type categoryScore struct {
		category string
		score    float64
	}

	sortedScores := make([]categoryScore, 0, len(scores))
	for cat, score := range scores {
		if cat != primary && score > 0.1 {
			sortedScores = append(sortedScores, categoryScore{cat, score})
		}
	}

	sort.Slice(sortedScores, func(i, j int) bool {
		return sortedScores[i].score > sortedScores[j].score
	})

	// Take top 2 alternatives
	alternatives := make([]types.ClassificationAlternative, 0, 2)
	for i := 0; i < len(sortedScores) && i < 2; i++ {
		cs := sortedScores[i]
		alternatives = append(alternatives, types.ClassificationAlternative{
			Category:   cs.category,
			Confidence: cs.score,
			Reason:     e.getAlternativeReason(cs.category, cs.score),
		})
	}

	return alternatives
}

// getAlternativeReason returns a reason for an alternative classification.
func (e *ClassifierEnginePro) getAlternativeReason(category string, score float64) string {
	switch category {
	case "task":
		return "Contains actionable language"
	case "habit":
		return "May indicate recurring activity"
	case "meeting":
		return "Contains meeting-related keywords"
	case "note":
		return "Could be informational content"
	case "event":
		return "May describe an event"
	default:
		return "Secondary match based on content analysis"
	}
}

// extractEntities extracts structured data from content.
func (e *ClassifierEnginePro) extractEntities(content string) types.ExtractedEntities {
	entities := types.ExtractedEntities{}

	// Extract title (first line or first 50 chars)
	lines := strings.Split(content, "\n")
	if len(lines) > 0 {
		title := strings.TrimSpace(lines[0])
		if len(title) > 80 {
			title = title[:80] + "..."
		}
		entities.Title = title
	}

	// Extract dates if enabled
	if e.getBool("extract_dates", true) {
		entities.DueDate = e.extractDate(content)
	}

	// Extract duration if enabled
	if e.getBool("extract_durations", true) {
		entities.Duration = e.extractDuration(content)
	}

	// Extract priority if enabled
	if e.getBool("extract_priorities", true) {
		entities.Priority = e.extractPriority(content)
	}

	// Extract people if enabled
	if e.getBool("extract_people", true) {
		entities.People = e.extractPeople(content)
	}

	// Extract URLs
	entities.URLs = e.extractURLs(content)

	// Extract tags
	entities.Tags = e.extractTags(content)

	return entities
}

// extractDate extracts date references from content.
func (e *ClassifierEnginePro) extractDate(content string) string {
	contentLower := strings.ToLower(content)

	// Common date patterns
	patterns := []struct {
		regex  string
		format string
	}{
		{`tomorrow`, "tomorrow"},
		{`today`, "today"},
		{`next (monday|tuesday|wednesday|thursday|friday|saturday|sunday)`, ""},
		{`(monday|tuesday|wednesday|thursday|friday|saturday|sunday)`, ""},
		{`end of (day|week|month)`, ""},
		{`by (monday|tuesday|wednesday|thursday|friday|saturday|sunday)`, ""},
		{`due (tomorrow|today|monday|tuesday|wednesday|thursday|friday|saturday|sunday)`, ""},
		{`\d{1,2}/\d{1,2}(/\d{2,4})?`, ""}, // MM/DD or MM/DD/YYYY
		{`\d{1,2}-\d{1,2}(-\d{2,4})?`, ""}, // MM-DD or MM-DD-YYYY
	}

	for _, p := range patterns {
		re := regexp.MustCompile(p.regex)
		if match := re.FindString(contentLower); match != "" {
			return match
		}
	}

	return ""
}

// extractDuration extracts time duration from content.
func (e *ClassifierEnginePro) extractDuration(content string) string {
	contentLower := strings.ToLower(content)

	patterns := []string{
		`(\d+)\s*(hour|hr|h)s?`,
		`(\d+)\s*(minute|min|m)s?`,
		`(\d+)\s*-\s*(\d+)\s*(hour|minute|min)s?`,
		`(half|quarter)\s*(hour|day)`,
		`(all day|full day)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindString(contentLower); match != "" {
			return match
		}
	}

	return ""
}

// extractPriority extracts priority indicators from content.
func (e *ClassifierEnginePro) extractPriority(content string) string {
	contentLower := strings.ToLower(content)

	urgentPatterns := []string{
		"urgent", "asap", "immediately", "critical", "high priority", "p1", "!important",
	}
	for _, p := range urgentPatterns {
		if strings.Contains(contentLower, p) {
			return "urgent"
		}
	}

	highPatterns := []string{
		"high", "important", "priority", "p2",
	}
	for _, p := range highPatterns {
		if strings.Contains(contentLower, p) {
			return "high"
		}
	}

	lowPatterns := []string{
		"low priority", "when possible", "nice to have", "optional", "p4",
	}
	for _, p := range lowPatterns {
		if strings.Contains(contentLower, p) {
			return "low"
		}
	}

	return ""
}

// extractPeople extracts names of people mentioned.
func (e *ClassifierEnginePro) extractPeople(content string) []string {
	people := make([]string, 0)

	// Pattern for "with [Name]" or "from [Name]"
	patterns := []string{
		`with ([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?)`,
		`from ([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?)`,
		`@([a-zA-Z]+)`,
		`([A-Z][a-z]+)'s\b`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				name := strings.TrimSpace(match[1])
				if len(name) > 1 && !isCommonWord(name) {
					people = append(people, name)
				}
			}
		}
	}

	return uniqueStrings(people)
}

// extractURLs extracts URLs from content.
func (e *ClassifierEnginePro) extractURLs(content string) []string {
	urlPattern := regexp.MustCompile(`https?://[^\s<>"]+`)
	return urlPattern.FindAllString(content, -1)
}

// extractTags extracts hashtags from content.
func (e *ClassifierEnginePro) extractTags(content string) []string {
	tagPattern := regexp.MustCompile(`#([a-zA-Z][a-zA-Z0-9_-]*)`)
	matches := tagPattern.FindAllStringSubmatch(content, -1)

	tags := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			tags = append(tags, match[1])
		}
	}

	return tags
}

// generateExplanation generates a human-readable explanation.
func (e *ClassifierEnginePro) generateExplanation(category string, confidence float64, content string) string {
	confidenceDesc := "high"
	if confidence < 0.5 {
		confidenceDesc = "low"
	} else if confidence < 0.7 {
		confidenceDesc = "moderate"
	}

	switch category {
	case "task":
		return "Classified as task with " + confidenceDesc + " confidence based on actionable language and imperative structure"
	case "habit":
		return "Classified as habit with " + confidenceDesc + " confidence based on recurring activity indicators"
	case "meeting":
		return "Classified as meeting with " + confidenceDesc + " confidence based on scheduling and interpersonal keywords"
	case "note":
		return "Classified as note with " + confidenceDesc + " confidence based on informational content markers"
	case "event":
		return "Classified as event with " + confidenceDesc + " confidence based on time-bound occurrence indicators"
	default:
		return "Classification determined with " + confidenceDesc + " confidence"
	}
}

// determineReviewReason explains why review is recommended.
func (e *ClassifierEnginePro) determineReviewReason(score float64, alternatives []types.ClassificationAlternative, content string) string {
	reasons := make([]string, 0)

	if score < 0.3 {
		reasons = append(reasons, "very low classification confidence")
	} else if score < 0.5 {
		reasons = append(reasons, "low classification confidence")
	}

	if len(alternatives) > 0 && alternatives[0].Confidence > score*0.8 {
		reasons = append(reasons, "close alternative classification exists")
	}

	if len(content) < 20 {
		reasons = append(reasons, "content is very short")
	}

	if len(reasons) == 0 {
		return "flagged for review based on classification threshold"
	}

	return strings.Join(reasons, "; ")
}

// Helper functions
func (e *ClassifierEnginePro) getFloat(key string, defaultVal float64) float64 {
	if e.config.Has(key) {
		return e.config.GetFloat(key)
	}
	return defaultVal
}

func (e *ClassifierEnginePro) getBool(key string, defaultVal bool) bool {
	if e.config.Has(key) {
		return e.config.GetBool(key)
	}
	return defaultVal
}

// isCommonWord checks if a word is too common to be a name.
func isCommonWord(word string) bool {
	common := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "from": true,
		"this": true, "that": true, "will": true, "have": true, "been": true,
		"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
		"friday": true, "saturday": true, "sunday": true,
	}
	return common[strings.ToLower(word)]
}

// uniqueStrings removes duplicates from a string slice.
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(strs))
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// Unused but keeping for potential future use
var _ = uuid.New
var _ = time.Now

// Ensure ClassifierEnginePro implements types.ClassifierEngine
var _ types.ClassifierEngine = (*ClassifierEnginePro)(nil)
