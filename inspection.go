package opskit

import "context"

// Inspection is a safe operational view of a component.
//
// Inspection is intended for admin endpoints, diagnostics, support workflows,
// and logs. It must not contain secrets, credentials, tokens, raw connection
// strings, or unredacted user data. Presentation layers such as Servekit may
// pass inspection data through directly, so components are responsible for
// redacting inspection data before returning it.
type Inspection struct {
	Summary    any         `json:"summary,omitempty"`
	Details    any         `json:"details,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
}

// Inspector reports safe operational inspection data.
//
// Components should implement Inspector only when they have useful diagnostic
// state beyond basic Status and Readiness. Inspection errors may be exposed in
// ComponentSnapshot.InspectionError, so returned errors must also be safe and
// redacted.
type Inspector interface {
	Inspect(context.Context) (Inspection, error)
}
