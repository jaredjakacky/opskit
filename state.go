package opskit

// State describes the high-level operational state of a component.
//
// State is intentionally generic. Domain-specific kits may expose richer
// internal states, but they should map them into this small shared vocabulary
// when reporting through Opskit.
type State string

const (
	StateUnknown      State = "unknown"
	StateInitializing State = "initializing"
	StateReady        State = "ready"
	StateDegraded     State = "degraded"
	StateNotReady     State = "not_ready"
	StateFailed       State = "failed"
	StateStopped      State = "stopped"
)
