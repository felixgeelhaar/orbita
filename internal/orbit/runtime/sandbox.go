package runtime

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/felixgeelhaar/orbita/internal/orbit/registry"
	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/google/uuid"
)

// Sandbox enforces capability restrictions for orbit execution.
type Sandbox struct {
	logger   *slog.Logger
	registry *registry.Registry

	// API factories (injected dependencies)
	taskAPIFactory     TaskAPIFactory
	habitAPIFactory    HabitAPIFactory
	scheduleAPIFactory ScheduleAPIFactory
	meetingAPIFactory  MeetingAPIFactory
	inboxAPIFactory    InboxAPIFactory
	storageAPIFactory  StorageAPIFactory
	metricsFactory     MetricsFactory
}

// TaskAPIFactory creates TaskAPI instances for users.
type TaskAPIFactory func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.TaskAPI

// HabitAPIFactory creates HabitAPI instances for users.
type HabitAPIFactory func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.HabitAPI

// ScheduleAPIFactory creates ScheduleAPI instances for users.
type ScheduleAPIFactory func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.ScheduleAPI

// MeetingAPIFactory creates MeetingAPI instances for users.
type MeetingAPIFactory func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.MeetingAPI

// InboxAPIFactory creates InboxAPI instances for users.
type InboxAPIFactory func(userID uuid.UUID, caps sdk.CapabilitySet) sdk.InboxAPI

// StorageAPIFactory creates StorageAPI instances for orbits.
type StorageAPIFactory func(orbitID string, userID uuid.UUID, caps sdk.CapabilitySet) sdk.StorageAPI

// MetricsFactory creates MetricsCollector instances for orbits.
type MetricsFactory func(orbitID string) sdk.MetricsCollector

// SandboxConfig holds configuration for the sandbox.
type SandboxConfig struct {
	Logger   *slog.Logger
	Registry *registry.Registry

	TaskAPIFactory     TaskAPIFactory
	HabitAPIFactory    HabitAPIFactory
	ScheduleAPIFactory ScheduleAPIFactory
	MeetingAPIFactory  MeetingAPIFactory
	InboxAPIFactory    InboxAPIFactory
	StorageAPIFactory  StorageAPIFactory
	MetricsFactory     MetricsFactory
}

// NewSandbox creates a new sandbox for orbit execution.
func NewSandbox(cfg SandboxConfig) *Sandbox {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Sandbox{
		logger:             cfg.Logger,
		registry:           cfg.Registry,
		taskAPIFactory:     cfg.TaskAPIFactory,
		habitAPIFactory:    cfg.HabitAPIFactory,
		scheduleAPIFactory: cfg.ScheduleAPIFactory,
		meetingAPIFactory:  cfg.MeetingAPIFactory,
		inboxAPIFactory:    cfg.InboxAPIFactory,
		storageAPIFactory:  cfg.StorageAPIFactory,
		metricsFactory:     cfg.MetricsFactory,
	}
}

// CreateContext creates a sandboxed OrbitContext for the given orbit and user.
func (s *Sandbox) CreateContext(
	ctx context.Context,
	orbitID string,
	userID uuid.UUID,
) (sdk.Context, error) {
	// Get orbit entry from registry
	manifest, err := s.registry.GetManifest(orbitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orbit manifest: %w", err)
	}

	// Parse capabilities from manifest
	capabilities, err := manifest.GetCapabilities()
	if err != nil {
		return nil, fmt.Errorf("failed to parse capabilities: %w", err)
	}
	capSet := sdk.NewCapabilitySet(capabilities)

	// Create capability-checked APIs
	cfg := OrbitContextConfig{
		OrbitID:      orbitID,
		UserID:       userID.String(),
		Capabilities: capSet,
		Logger:       s.logger,
	}

	// Only provide APIs for declared capabilities
	if capSet.Has(sdk.CapReadTasks) && s.taskAPIFactory != nil {
		cfg.TaskAPI = s.taskAPIFactory(userID, capSet)
	}
	if capSet.Has(sdk.CapReadHabits) && s.habitAPIFactory != nil {
		cfg.HabitAPI = s.habitAPIFactory(userID, capSet)
	}
	if capSet.Has(sdk.CapReadSchedule) && s.scheduleAPIFactory != nil {
		cfg.ScheduleAPI = s.scheduleAPIFactory(userID, capSet)
	}
	if capSet.Has(sdk.CapReadMeetings) && s.meetingAPIFactory != nil {
		cfg.MeetingAPI = s.meetingAPIFactory(userID, capSet)
	}
	if capSet.Has(sdk.CapReadInbox) && s.inboxAPIFactory != nil {
		cfg.InboxAPI = s.inboxAPIFactory(userID, capSet)
	}
	if (capSet.Has(sdk.CapReadStorage) || capSet.Has(sdk.CapWriteStorage)) && s.storageAPIFactory != nil {
		cfg.StorageAPI = s.storageAPIFactory(orbitID, userID, capSet)
	}
	if s.metricsFactory != nil {
		cfg.Metrics = s.metricsFactory(orbitID)
	}

	s.logger.Info("created sandboxed context",
		"orbit_id", orbitID,
		"user_id", userID.String(),
		"capabilities", capabilities,
	)

	return NewOrbitContext(ctx, cfg), nil
}

// ValidateCapabilities validates that an orbit's declared capabilities match its requirements.
func (s *Sandbox) ValidateCapabilities(orbitID string) error {
	return s.registry.ValidateCapabilities(orbitID)
}

// CheckCapability checks if an orbit has a specific capability.
func (s *Sandbox) CheckCapability(orbitID string, cap sdk.Capability) (bool, error) {
	manifest, err := s.registry.GetManifest(orbitID)
	if err != nil {
		return false, err
	}

	capabilities, err := manifest.GetCapabilities()
	if err != nil {
		return false, err
	}

	capSet := sdk.NewCapabilitySet(capabilities)
	return capSet.Has(cap), nil
}

// Executor orchestrates orbit execution with proper sandboxing.
type Executor struct {
	sandbox  *Sandbox
	registry *registry.Registry
	logger   *slog.Logger
}

// ExecutorConfig holds configuration for the executor.
type ExecutorConfig struct {
	Sandbox  *Sandbox
	Registry *registry.Registry
	Logger   *slog.Logger
}

// NewExecutor creates a new orbit executor.
func NewExecutor(cfg ExecutorConfig) *Executor {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Executor{
		sandbox:  cfg.Sandbox,
		registry: cfg.Registry,
		logger:   cfg.Logger,
	}
}

// InitializeOrbit initializes an orbit with a sandboxed context.
func (e *Executor) InitializeOrbit(
	ctx context.Context,
	orbitID string,
	userID uuid.UUID,
) error {
	// Get orbit from registry
	orbit, err := e.registry.Get(ctx, orbitID, userID)
	if err != nil {
		return fmt.Errorf("failed to get orbit: %w", err)
	}

	// Create sandboxed context
	orbitCtx, err := e.sandbox.CreateContext(ctx, orbitID, userID)
	if err != nil {
		return fmt.Errorf("failed to create sandbox context: %w", err)
	}

	// Initialize orbit with sandboxed context
	if err := orbit.Initialize(orbitCtx); err != nil {
		return fmt.Errorf("failed to initialize orbit: %w", err)
	}

	e.logger.Info("initialized orbit",
		"orbit_id", orbitID,
		"user_id", userID.String(),
	)

	return nil
}

// ShutdownOrbit shuts down an orbit.
func (e *Executor) ShutdownOrbit(ctx context.Context, orbitID string) error {
	// Registry handles shutdown
	return e.registry.Unregister(ctx, orbitID)
}

// GetOrbit returns an orbit instance for direct access.
func (e *Executor) GetOrbit(
	ctx context.Context,
	orbitID string,
	userID uuid.UUID,
) (sdk.Orbit, error) {
	return e.registry.Get(ctx, orbitID, userID)
}
