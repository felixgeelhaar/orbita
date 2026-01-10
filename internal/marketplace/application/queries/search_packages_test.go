package queries

import (
	"context"
	"errors"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSearchPackagesHandler_Handle(t *testing.T) {
	t.Run("successfully searches packages", func(t *testing.T) {
		mockRepo := new(MockPackageRepository)
		handler := NewSearchPackagesHandler(mockRepo)

		pkg := createTestPackage("test.orbit", "Test Orbit", domain.PackageTypeOrbit)
		packages := []*domain.Package{pkg}

		mockRepo.On("Search", mock.Anything, "test", mock.MatchedBy(func(filter domain.PackageFilter) bool {
			return filter.Limit == 20
		})).Return(packages, int64(1), nil)

		query := SearchPackagesQuery{
			Query:  "test",
			Offset: 0,
			Limit:  20,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Packages, 1)
		assert.Equal(t, int64(1), result.Total)
		assert.Equal(t, "test", result.Query)
		mockRepo.AssertExpectations(t)
	})

	t.Run("uses default limit when not specified", func(t *testing.T) {
		mockRepo := new(MockPackageRepository)
		handler := NewSearchPackagesHandler(mockRepo)

		mockRepo.On("Search", mock.Anything, "query", mock.MatchedBy(func(filter domain.PackageFilter) bool {
			return filter.Limit == 20
		})).Return([]*domain.Package{}, int64(0), nil)

		query := SearchPackagesQuery{
			Query:  "query",
			Offset: 0,
			Limit:  0, // Should default to 20
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("applies type filter", func(t *testing.T) {
		mockRepo := new(MockPackageRepository)
		handler := NewSearchPackagesHandler(mockRepo)

		engineType := domain.PackageTypeEngine
		pkg := createTestPackage("test.engine", "Test Engine", domain.PackageTypeEngine)

		mockRepo.On("Search", mock.Anything, "engine", mock.MatchedBy(func(filter domain.PackageFilter) bool {
			return filter.Type != nil && *filter.Type == domain.PackageTypeEngine
		})).Return([]*domain.Package{pkg}, int64(1), nil)

		query := SearchPackagesQuery{
			Query: "engine",
			Type:  &engineType,
			Limit: 20,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, result.Packages, 1)
		assert.Equal(t, "engine", result.Packages[0].Type)
		mockRepo.AssertExpectations(t)
	})

	t.Run("applies sort options", func(t *testing.T) {
		mockRepo := new(MockPackageRepository)
		handler := NewSearchPackagesHandler(mockRepo)

		mockRepo.On("Search", mock.Anything, "test", mock.MatchedBy(func(filter domain.PackageFilter) bool {
			return filter.SortBy == domain.SortByRating && filter.SortOrder == domain.SortDesc
		})).Return([]*domain.Package{}, int64(0), nil)

		query := SearchPackagesQuery{
			Query:    "test",
			SortBy:   domain.SortByRating,
			SortDesc: true,
			Limit:    20,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("applies verified filter", func(t *testing.T) {
		mockRepo := new(MockPackageRepository)
		handler := NewSearchPackagesHandler(mockRepo)

		verified := true
		mockRepo.On("Search", mock.Anything, "verified", mock.MatchedBy(func(filter domain.PackageFilter) bool {
			return filter.Verified != nil && *filter.Verified == true
		})).Return([]*domain.Package{}, int64(0), nil)

		query := SearchPackagesQuery{
			Query:    "verified",
			Verified: &verified,
			Limit:    20,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("fails when repository returns error", func(t *testing.T) {
		mockRepo := new(MockPackageRepository)
		handler := NewSearchPackagesHandler(mockRepo)

		mockRepo.On("Search", mock.Anything, "test", mock.Anything).Return(nil, int64(0), errors.New("database error"))

		query := SearchPackagesQuery{
			Query: "test",
			Limit: 20,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns empty result for no matches", func(t *testing.T) {
		mockRepo := new(MockPackageRepository)
		handler := NewSearchPackagesHandler(mockRepo)

		mockRepo.On("Search", mock.Anything, "nonexistent", mock.Anything).Return([]*domain.Package{}, int64(0), nil)

		query := SearchPackagesQuery{
			Query: "nonexistent",
			Limit: 20,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Packages)
		assert.Equal(t, int64(0), result.Total)
		mockRepo.AssertExpectations(t)
	})
}

func TestNewSearchPackagesHandler(t *testing.T) {
	mockRepo := new(MockPackageRepository)

	handler := NewSearchPackagesHandler(mockRepo)

	assert.NotNil(t, handler)
}
