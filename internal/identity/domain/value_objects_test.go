package domain_test

import (
	"strings"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/identity/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
		want    string
	}{
		{"valid email", "user@example.com", nil, "user@example.com"},
		{"uppercase", "USER@EXAMPLE.COM", nil, "user@example.com"},
		{"with spaces", "  user@example.com  ", nil, "user@example.com"},
		{"with plus", "user+tag@example.com", nil, "user+tag@example.com"},
		{"subdomain", "user@sub.example.com", nil, "user@sub.example.com"},
		{"empty", "", domain.ErrInvalidEmail, ""},
		{"no @", "userexample.com", domain.ErrInvalidEmail, ""},
		{"no domain", "user@", domain.ErrInvalidEmail, ""},
		{"no local part", "@example.com", domain.ErrInvalidEmail, ""},
		{"multiple @", "user@@example.com", domain.ErrInvalidEmail, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := domain.NewEmail(tt.input)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, email.String())
			}
		})
	}
}

func TestEmail_Domain(t *testing.T) {
	email, _ := domain.NewEmail("user@example.com")
	assert.Equal(t, "example.com", email.Domain())
}

func TestEmail_Equals(t *testing.T) {
	email1, _ := domain.NewEmail("user@example.com")
	email2, _ := domain.NewEmail("USER@EXAMPLE.COM")
	email3, _ := domain.NewEmail("other@example.com")

	assert.True(t, email1.Equals(email2))
	assert.False(t, email1.Equals(email3))
}

func TestNewName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
		want    string
	}{
		{"valid name", "John Doe", nil, "John Doe"},
		{"with spaces", "  John Doe  ", nil, "John Doe"},
		{"single name", "John", nil, "John"},
		{"empty", "", domain.ErrEmptyName, ""},
		{"whitespace only", "   ", domain.ErrEmptyName, ""},
		{"too long", strings.Repeat("a", 256), domain.ErrNameTooLong, ""},
		{"max length", strings.Repeat("a", 255), nil, strings.Repeat("a", 255)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := domain.NewName(tt.input)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, name.String())
			}
		})
	}
}

func TestName_Equals(t *testing.T) {
	name1, _ := domain.NewName("John Doe")
	name2, _ := domain.NewName("John Doe")
	name3, _ := domain.NewName("Jane Doe")

	assert.True(t, name1.Equals(name2))
	assert.False(t, name1.Equals(name3))
}
