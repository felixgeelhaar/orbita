package queries

import (
	"context"
	"errors"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetFeaturedHandler_Handle(t *testing.T) {
	t.Run("successfully gets featured packages", func(t *testing.T) {
		mockRepo := new(MockPackageRepository)
		handler := NewGetFeaturedHandler(mockRepo)

		pkg1 := createTestPackage("featured.orbit1", "Featured Orbit 1", domain.PackageTypeOrbit)
		pkg1.Featured = true
		pkg2 := createTestPackage("featured.orbit2", "Featured Orbit 2", domain.PackageTypeOrbit)
		pkg2.Featured = true
		packages := []*domain.Package{pkg1, pkg2}

		mockRepo.On("GetFeatured", mock.Anything, 10).Return(packages, nil)

		query := GetFeaturedQuery{
			Limit: 10,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Packages, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("uses default limit when not specified", func(t *testing.T) {
		mockRepo := new(MockPackageRepository)
		handler := NewGetFeaturedHandler(mockRepo)

		mockRepo.On("GetFeatured", mock.Anything, 10).Return([]*domain.Package{}, nil)

		query := GetFeaturedQuery{
			Limit: 0, // Should default to 10
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("respects custom limit", func(t *testing.T) {
		mockRepo := new(MockPackageRepository)
		handler := NewGetFeaturedHandler(mockRepo)

		mockRepo.On("GetFeatured", mock.Anything, 5).Return([]*domain.Package{}, nil)

		query := GetFeaturedQuery{
			Limit: 5,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("fails when repository returns error", func(t *testing.T) {
		mockRepo := new(MockPackageRepository)
		handler := NewGetFeaturedHandler(mockRepo)

		mockRepo.On("GetFeatured", mock.Anything, 10).Return(nil, errors.New("database error"))

		query := GetFeaturedQuery{
			Limit: 10,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns empty list when no featured packages", func(t *testing.T) {
		mockRepo := new(MockPackageRepository)
		handler := NewGetFeaturedHandler(mockRepo)

		mockRepo.On("GetFeatured", mock.Anything, 10).Return([]*domain.Package{}, nil)

		query := GetFeaturedQuery{
			Limit: 10,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Packages)
		mockRepo.AssertExpectations(t)
	})
}

func TestNewGetFeaturedHandler(t *testing.T) {
	mockRepo := new(MockPackageRepository)

	handler := NewGetFeaturedHandler(mockRepo)

	assert.NotNil(t, handler)
}
