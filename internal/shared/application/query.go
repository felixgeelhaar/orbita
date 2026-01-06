package application

import "context"

// Query represents a query that reads system state.
type Query interface {
	QueryName() string
}

// QueryHandler handles a specific query type.
type QueryHandler[Q Query, R any] interface {
	Handle(ctx context.Context, query Q) (R, error)
}
