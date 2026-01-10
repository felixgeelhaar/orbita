package sdk

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadata_Validate(t *testing.T) {
	t.Run("valid metadata passes validation", func(t *testing.T) {
		m := Metadata{
			ID:      "acme.wellness",
			Name:    "ACME Wellness",
			Version: "1.0.0",
		}

		err := m.Validate()

		assert.NoError(t, err)
	})

	t.Run("valid metadata with optional fields", func(t *testing.T) {
		m := Metadata{
			ID:            "acme.wellness",
			Name:          "ACME Wellness",
			Version:       "1.0.0",
			Author:        "ACME Corp",
			Description:   "Wellness tracking orbit",
			License:       "MIT",
			Homepage:      "https://acme.example.com",
			Tags:          []string{"wellness", "health"},
			MinAPIVersion: "1.0.0",
			RequiredOrbit: "orbita.base",
		}

		err := m.Validate()

		assert.NoError(t, err)
	})

	t.Run("returns ErrMissingID for empty ID", func(t *testing.T) {
		m := Metadata{
			Name:    "ACME Wellness",
			Version: "1.0.0",
		}

		err := m.Validate()

		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrMissingID))
	})

	t.Run("returns ErrMissingName for empty Name", func(t *testing.T) {
		m := Metadata{
			ID:      "acme.wellness",
			Version: "1.0.0",
		}

		err := m.Validate()

		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrMissingName))
	})

	t.Run("returns ErrMissingVersion for empty Version", func(t *testing.T) {
		m := Metadata{
			ID:   "acme.wellness",
			Name: "ACME Wellness",
		}

		err := m.Validate()

		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrMissingVersion))
	})

	t.Run("validation checks fields in order", func(t *testing.T) {
		// Empty ID is checked first
		m := Metadata{}
		err := m.Validate()
		assert.True(t, errors.Is(err, ErrMissingID))

		// Then Name
		m.ID = "test"
		err = m.Validate()
		assert.True(t, errors.Is(err, ErrMissingName))

		// Then Version
		m.Name = "Test"
		err = m.Validate()
		assert.True(t, errors.Is(err, ErrMissingVersion))

		// Finally passes
		m.Version = "1.0.0"
		err = m.Validate()
		assert.NoError(t, err)
	})
}

func TestMetadata_String(t *testing.T) {
	t.Run("returns ID@Version format", func(t *testing.T) {
		m := Metadata{
			ID:      "acme.wellness",
			Name:    "ACME Wellness",
			Version: "2.1.3",
		}

		assert.Equal(t, "acme.wellness@2.1.3", m.String())
	})

	t.Run("works with empty fields", func(t *testing.T) {
		m := Metadata{}

		assert.Equal(t, "@", m.String())
	})

	t.Run("formats various version formats", func(t *testing.T) {
		tests := []struct {
			id      string
			version string
			want    string
		}{
			{"test.orbit", "1.0.0", "test.orbit@1.0.0"},
			{"orbita.wellness", "0.0.1", "orbita.wellness@0.0.1"},
			{"vendor.long-name", "10.20.30", "vendor.long-name@10.20.30"},
		}

		for _, tc := range tests {
			m := Metadata{ID: tc.id, Version: tc.version}
			assert.Equal(t, tc.want, m.String())
		}
	})
}
