// Package opskit defines small operational contracts for Go services.
//
// Opskit is the shared operational vocabulary for the Kit Series. It does not
// own HTTP serving, worker execution, configuration loading, outbound clients,
// dependency checks, persistence, telemetry backends, or application policy.
//
// Other kits implement Opskit contracts so they can be registered, inspected,
// checked, controlled, and exposed consistently by service bootstrap code.
//
// Opskit may define passive data shapes, small interfaces, and a passive
// component registry. It must not execute checks, dispatch commands, schedule
// work, authorize operations, export telemetry, own lifecycle, or decide
// application policy.
//
// Registry methods are safe for concurrent use, but component implementations
// may be called concurrently by different callers. Components that expose
// mutable state through Opskit interfaces are responsible for their own
// synchronization.
package opskit
