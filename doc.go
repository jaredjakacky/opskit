// Package opskit defines small operational contracts for Go services.
//
// Opskit is the shared operational vocabulary for the Kit Series. It does not
// own HTTP serving, worker execution, configuration loading, outbound clients,
// dependency checks, persistence, telemetry backends, or application policy.
//
// Other kits implement Opskit contracts so application assembly can register
// them, presentation layers can expose descriptive state, and execution layers
// can explicitly invoke selected active capabilities.
//
// Opskit may define data shapes, small interfaces, capability metadata, and a
// passive component registry. It must not execute checks, dispatch commands,
// schedule work, authorize operations, export telemetry, own lifecycle, or
// decide application policy.
//
// Registry methods are safe for concurrent use, but component implementations
// may be called concurrently by different callers. Components that expose
// mutable state through Opskit interfaces are responsible for their own
// synchronization.
package opskit
