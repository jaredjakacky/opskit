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
package opskit
