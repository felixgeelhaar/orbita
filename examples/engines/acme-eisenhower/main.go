// ACME Eisenhower Matrix Priority Engine
//
// This is an example third-party engine plugin demonstrating how to build
// custom priority engines for Orbita using the public enginesdk package.
//
// The engine implements the Eisenhower Matrix method for task prioritization,
// categorizing tasks into four quadrants based on urgency and importance.
//
// Usage:
//
//	# Build the plugin
//	go build -o acme-eisenhower-engine ./examples/engines/acme-eisenhower
//
//	# The plugin is loaded automatically by Orbita's engine loader
package main

import (
	"github.com/felixgeelhaar/orbita/pkg/enginesdk"
)

func main() {
	// Create the engine instance
	engine := New()

	// Serve the plugin using the SDK helper
	// This starts the gRPC server and handles the go-plugin protocol
	enginesdk.ServePriority(engine)
}
