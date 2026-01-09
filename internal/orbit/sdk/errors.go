package sdk

import "errors"

// SDK error types for orbit development.
var (
	// Metadata validation errors
	ErrMissingID      = errors.New("orbit metadata: missing ID")
	ErrMissingName    = errors.New("orbit metadata: missing name")
	ErrMissingVersion = errors.New("orbit metadata: missing version")

	// Capability errors
	ErrInvalidCapability    = errors.New("invalid capability")
	ErrCapabilityNotGranted = errors.New("capability not granted")
	ErrCapabilityMismatch   = errors.New("declared capabilities do not match required capabilities")

	// Lifecycle errors
	ErrOrbitNotInitialized = errors.New("orbit not initialized")
	ErrOrbitAlreadyLoaded  = errors.New("orbit already loaded")
	ErrOrbitNotFound       = errors.New("orbit not found")
	ErrOrbitNotEntitled    = errors.New("user not entitled to this orbit")

	// Registration errors
	ErrToolAlreadyRegistered    = errors.New("tool already registered")
	ErrCommandAlreadyRegistered = errors.New("command already registered")
	ErrInvalidToolName          = errors.New("invalid tool name")
	ErrInvalidCommandName       = errors.New("invalid command name")

	// Storage errors
	ErrStorageKeyNotFound = errors.New("storage key not found")
	ErrStorageKeyTooLong  = errors.New("storage key too long")
	ErrStorageValueTooBig = errors.New("storage value too big")

	// Event errors
	ErrInvalidEventType   = errors.New("invalid event type")
	ErrEventHandlerFailed = errors.New("event handler failed")

	// API errors
	ErrResourceNotFound = errors.New("resource not found")
	ErrAccessDenied     = errors.New("access denied")

	// Manifest errors
	ErrManifestNotFound    = errors.New("orbit manifest not found")
	ErrManifestInvalid     = errors.New("orbit manifest is invalid")
	ErrManifestMissingID   = errors.New("orbit manifest missing ID")
	ErrManifestMissingType = errors.New("orbit manifest missing type")

	// Version errors
	ErrIncompatibleAPIVersion = errors.New("incompatible API version")
)

// OrbitError wraps an error with orbit context.
type OrbitError struct {
	OrbitID string
	Op      string
	Err     error
}

func (e *OrbitError) Error() string {
	if e.OrbitID != "" {
		return "orbit " + e.OrbitID + ": " + e.Op + ": " + e.Err.Error()
	}
	return e.Op + ": " + e.Err.Error()
}

func (e *OrbitError) Unwrap() error {
	return e.Err
}

// NewOrbitError creates a new OrbitError.
func NewOrbitError(orbitID, op string, err error) *OrbitError {
	return &OrbitError{
		OrbitID: orbitID,
		Op:      op,
		Err:     err,
	}
}

// CapabilityError represents an error when a capability check fails.
type CapabilityError struct {
	OrbitID    string
	Capability Capability
	Operation  string
}

func (e *CapabilityError) Error() string {
	return "orbit " + e.OrbitID + ": capability " + string(e.Capability) + " required for " + e.Operation
}

// NewCapabilityError creates a new CapabilityError.
func NewCapabilityError(orbitID string, cap Capability, op string) *CapabilityError {
	return &CapabilityError{
		OrbitID:    orbitID,
		Capability: cap,
		Operation:  op,
	}
}
