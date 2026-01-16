package domain

import "errors"

var (
	// ErrLicenseNotFound indicates no license file exists.
	ErrLicenseNotFound = errors.New("license not found")

	// ErrLicenseExpired indicates the license has expired beyond the grace period.
	ErrLicenseExpired = errors.New("license expired")

	// ErrInvalidSignature indicates the license signature verification failed.
	ErrInvalidSignature = errors.New("invalid license signature")

	// ErrInvalidLicenseKey indicates the license key format is invalid.
	ErrInvalidLicenseKey = errors.New("invalid license key format")

	// ErrLicenseRevoked indicates the license has been revoked by the server.
	ErrLicenseRevoked = errors.New("license has been revoked")

	// ErrActivationFailed indicates the activation request failed.
	ErrActivationFailed = errors.New("license activation failed")

	// ErrNetworkError indicates a network error during license validation.
	ErrNetworkError = errors.New("network error during license validation")
)
