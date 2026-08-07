package opskit

// FailureCodeInspectionFailed is the stable public code used when an inspector
// fails while Registry.Snapshot is building a component snapshot.
const FailureCodeInspectionFailed = "inspection_failed"

// Failure is safe public operational failure detail.
//
// Code should be a stable, low-cardinality token suitable for programmatic
// handling. Message should contain only bounded, redacted text that is safe for
// logs, admin endpoints, dashboards, diagnostics, support tools, and tests.
//
// Failure deliberately does not retain an underlying error. The implementing
// component or domain kit must retain private causes through its native API,
// internal state, or private observation path before projecting an Opskit
// result.
type Failure struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func failurePtr(failure Failure) *Failure {
	if failure == (Failure{}) {
		return nil
	}

	copy := failure
	return &copy
}
