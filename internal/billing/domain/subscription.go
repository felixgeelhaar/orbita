package domain

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus represents the current billing state.
type SubscriptionStatus string

const (
	SubscriptionActive   SubscriptionStatus = "active"
	SubscriptionTrialing SubscriptionStatus = "trialing"
	SubscriptionPastDue  SubscriptionStatus = "past_due"
	SubscriptionCanceled SubscriptionStatus = "canceled"
)

// Subscription represents a user's subscription.
type Subscription struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	Plan                 string
	Status               SubscriptionStatus
	CurrentPeriodEnd     *time.Time
	StripeCustomerID     string
	StripeSubscriptionID string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
