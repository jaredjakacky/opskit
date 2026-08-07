package opskit

import "context"

// Inspection is a safe operational view of a component.
//
// Inspection is intended for admin endpoints, diagnostics, support workflows,
// and logs. It must not contain secrets, credentials, tokens, raw connection
// strings, or unredacted user data. Presentation layers such as Servekit may
// pass inspection data through directly, so components are responsible for
// redacting inspection data before returning it.
//
// Summary and Details must be JSON-marshalable values. Prefer strings,
// numbers, booleans, nil, slices, maps with string keys, or structs with
// stable JSON tags. Do not return functions, channels, cyclic values,
// non-finite floats, or values that require unavailable custom encoders.
type Inspection struct {
	Summary    any         `json:"summary,omitempty"`
	Details    any         `json:"details,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
}

// Inspector reports safe operational inspection data.
//
// Components should implement Inspector only when they have useful diagnostic
// state beyond basic Status and Readiness. Registry.Snapshot replaces
// inspection errors with generic public failure detail; Registry.Inspect
// returns the original error directly to its caller.
//
// Inspect is a descriptive hook used on administrative request paths. It should
// read bounded local or cached state and must not run checks, dispatch commands,
// call external services, or mutate lifecycle state.
type Inspector interface {
	Inspect(context.Context) (Inspection, error)
}
