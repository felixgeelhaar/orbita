package mcp

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	dateLayout = "2006-01-02"
	timeLayout = "15:04"
)

func parseDate(value string, fallback time.Time) (time.Time, error) {
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
	}
	return parsed, nil
}

func parseTimeOnDate(date time.Time, value string) (time.Time, error) {
	parsed, err := time.Parse(timeLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format, use HH:MM: %w", err)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), 0, 0, date.Location()), nil
}

func parseUUID(value string) (uuid.UUID, error) {
	if value == "" {
		return uuid.UUID{}, errors.New("id is required")
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid id: %w", err)
	}
	return id, nil
}

func parseOptionalUUID(value string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, nil
	}
	return parseUUID(value)
}

func parseOptionalTime(date time.Time, value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := parseTimeOnDate(date, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
