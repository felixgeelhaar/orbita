package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/orbita/internal/engine/grpc"
	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// Loader handles loading plugin engines using HashiCorp go-plugin.
type Loader struct {
	logger  *slog.Logger
	clients map[string]*plugin.Client
}

// NewLoader creates a new plugin loader.
func NewLoader(logger *slog.Logger) *Loader {
	if logger == nil {
		logger = slog.Default()
	}
	return &Loader{
		logger:  logger,
		clients: make(map[string]*plugin.Client),
	}
}

// LoadOptions contains options for loading a plugin.
type LoadOptions struct {
	// Manifest is the plugin manifest.
	Manifest *Manifest

	// Config is the initial configuration for the engine.
	Config sdk.EngineConfig

	// SecureMode enables additional security checks.
	SecureMode bool
}

// Load loads a plugin engine from a binary.
func (l *Loader) Load(ctx context.Context, opts LoadOptions) (sdk.Engine, error) {
	if opts.Manifest == nil {
		return nil, fmt.Errorf("manifest is required")
	}

	manifest := opts.Manifest
	binaryPath := manifest.BinaryAbsPath()

	// Security: Validate and sanitize the binary path before execution
	sanitizedPath, err := l.validateBinaryPath(binaryPath)
	if err != nil {
		return nil, sdk.NewLoadError(binaryPath, "binary path validation failed", err)
	}

	// Verify binary exists and is executable
	info, err := os.Stat(sanitizedPath)
	if err != nil {
		return nil, sdk.NewLoadError(sanitizedPath, "binary not found", err)
	}

	// Security: Ensure it's a regular file, not a directory or symlink
	if !info.Mode().IsRegular() {
		return nil, sdk.NewLoadError(sanitizedPath, "binary path is not a regular file", nil)
	}

	// Verify checksum if provided
	if opts.SecureMode && manifest.Checksum != "" {
		if err := l.verifyChecksum(sanitizedPath, manifest.Checksum); err != nil {
			return nil, sdk.NewLoadError(sanitizedPath, "checksum verification failed", err)
		}
	}

	l.logger.Info("loading plugin",
		"engine_id", manifest.ID,
		"binary", sanitizedPath,
	)

	// Configure the plugin client
	// #nosec G204 -- binary path is validated by validateBinaryPath
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: grpc.HandshakeConfig,
		Plugins:         grpc.PluginMapForEngine(manifest.EngineType()),
		Cmd:             exec.Command(sanitizedPath),
		Logger:          newHclogAdapter(l.logger),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
	})

	// Connect to the plugin
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, sdk.NewLoadError(binaryPath, "failed to connect", err)
	}

	// Dispense the engine plugin
	raw, err := rpcClient.Dispense("engine")
	if err != nil {
		client.Kill()
		return nil, sdk.NewLoadError(binaryPath, "failed to dispense", err)
	}

	engine, ok := raw.(sdk.Engine)
	if !ok {
		client.Kill()
		return nil, sdk.NewLoadError(binaryPath, "plugin does not implement Engine interface", nil)
	}

	// Initialize the engine
	if err := engine.Initialize(ctx, opts.Config); err != nil {
		client.Kill()
		return nil, sdk.NewLoadError(binaryPath, "initialization failed", err)
	}

	// Store the client for cleanup
	l.clients[manifest.ID] = client

	l.logger.Info("plugin loaded successfully",
		"engine_id", manifest.ID,
		"type", engine.Type(),
	)

	return engine, nil
}

// Unload stops and cleans up a plugin.
func (l *Loader) Unload(id string) error {
	client, exists := l.clients[id]
	if !exists {
		return nil // Already unloaded
	}

	client.Kill()
	delete(l.clients, id)

	l.logger.Info("plugin unloaded", "engine_id", id)
	return nil
}

// UnloadAll stops and cleans up all plugins.
func (l *Loader) UnloadAll() {
	for id, client := range l.clients {
		client.Kill()
		l.logger.Info("plugin unloaded", "engine_id", id)
	}
	l.clients = make(map[string]*plugin.Client)
}

// IsLoaded checks if a plugin is currently loaded.
func (l *Loader) IsLoaded(id string) bool {
	_, exists := l.clients[id]
	return exists
}

// validateBinaryPath validates and sanitizes a binary path to prevent command injection.
// It ensures the path is absolute, contains no shell metacharacters, and resolves
// to a safe location without path traversal.
func (l *Loader) validateBinaryPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("binary path cannot be empty")
	}

	// Clean the path to resolve any ".." or "." components
	cleanPath := filepath.Clean(path)

	// Ensure the path is absolute
	if !filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("binary path must be absolute: %s", path)
	}

	// Security: Check for shell metacharacters that could enable command injection
	// These characters have special meaning in shells and could be exploited
	dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "{", "}", "<", ">", "!", "\n", "\r", "\\", "'", "\""}
	for _, char := range dangerousChars {
		if strings.Contains(cleanPath, char) {
			return "", fmt.Errorf("binary path contains forbidden character %q: %s", char, path)
		}
	}

	// Security: Ensure the cleaned path doesn't escape via symlink by evaluating it
	// This resolves symlinks to their actual targets
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		// If the file doesn't exist yet, filepath.EvalSymlinks will fail
		// In that case, we return the cleaned path and let the caller handle existence check
		if os.IsNotExist(err) {
			return cleanPath, nil
		}
		return "", fmt.Errorf("failed to resolve binary path: %w", err)
	}

	l.logger.Debug("binary path validated",
		"original", path,
		"resolved", resolvedPath,
	)

	return resolvedPath, nil
}

// verifyChecksum verifies the SHA256 checksum of a file.
// Expected format: "sha256:HEXHASH" or just "HEXHASH" (assumes sha256)
func (l *Loader) verifyChecksum(path, expected string) error {
	// Parse expected checksum format
	algorithm := "sha256"
	hash := expected

	if strings.Contains(expected, ":") {
		parts := strings.SplitN(expected, ":", 2)
		algorithm = strings.ToLower(parts[0])
		hash = parts[1]
	}

	// Currently only SHA256 is supported
	if algorithm != "sha256" {
		return fmt.Errorf("unsupported checksum algorithm: %s (only sha256 is supported)", algorithm)
	}

	// Open the file
	// #nosec G304 - path is validated by validateBinaryPath before calling verifyChecksum
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Compute SHA256 hash
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	computed := hex.EncodeToString(hasher.Sum(nil))

	// Compare (case-insensitive)
	if !strings.EqualFold(computed, hash) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", hash, computed)
	}

	l.logger.Debug("checksum verified",
		"path", path,
		"algorithm", algorithm,
	)

	return nil
}

// hclogAdapter adapts slog to hclog interface
type hclogAdapter struct {
	logger *slog.Logger
	name   string
}

func newHclogAdapter(logger *slog.Logger) *hclogAdapter {
	return &hclogAdapter{logger: logger, name: "orbita"}
}

// Implement hclog.Logger interface
func (h *hclogAdapter) Log(level hclog.Level, msg string, args ...interface{}) {
	switch level {
	case hclog.Trace, hclog.Debug:
		h.logger.Debug(msg, args...)
	case hclog.Info:
		h.logger.Info(msg, args...)
	case hclog.Warn:
		h.logger.Warn(msg, args...)
	case hclog.Error:
		h.logger.Error(msg, args...)
	default:
		h.logger.Debug(msg, args...)
	}
}

func (h *hclogAdapter) Trace(msg string, args ...interface{}) {
	h.logger.Debug(msg, args...)
}

func (h *hclogAdapter) Debug(msg string, args ...interface{}) {
	h.logger.Debug(msg, args...)
}

func (h *hclogAdapter) Info(msg string, args ...interface{}) {
	h.logger.Info(msg, args...)
}

func (h *hclogAdapter) Warn(msg string, args ...interface{}) {
	h.logger.Warn(msg, args...)
}

func (h *hclogAdapter) Error(msg string, args ...interface{}) {
	h.logger.Error(msg, args...)
}

func (h *hclogAdapter) IsTrace() bool { return false }
func (h *hclogAdapter) IsDebug() bool { return true }
func (h *hclogAdapter) IsInfo() bool  { return true }
func (h *hclogAdapter) IsWarn() bool  { return true }
func (h *hclogAdapter) IsError() bool { return true }

func (h *hclogAdapter) ImpliedArgs() []interface{} { return nil }

func (h *hclogAdapter) With(args ...interface{}) hclog.Logger {
	return h
}

func (h *hclogAdapter) Name() string { return h.name }

func (h *hclogAdapter) Named(name string) hclog.Logger {
	return &hclogAdapter{
		logger: h.logger,
		name:   h.name + "." + name,
	}
}

func (h *hclogAdapter) ResetNamed(name string) hclog.Logger {
	return &hclogAdapter{
		logger: h.logger,
		name:   name,
	}
}

func (h *hclogAdapter) SetLevel(level hclog.Level) {}

func (h *hclogAdapter) GetLevel() hclog.Level { return hclog.Debug }

func (h *hclogAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return log.Default()
}

func (h *hclogAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return os.Stderr
}
