package opskit

import "context"

// ComponentInfo identifies one operational component.
//
// Name must be stable and unique within a Registry. Names must be safe single
// path segments containing only ASCII letters, ASCII digits, dots, underscores,
// and hyphens. Kind should be a low-cardinality category such as "config",
// "worker_runtime", "clients", "dependencies", or "state". Kind is not
// validated by Opskit; prefer stable, safe tokens because presentation and
// telemetry layers may use it in filters, labels, dashboards, or routes.
type ComponentInfo struct {
	Name        string `json:"name"`
	Kind        string `json:"kind,omitempty"`
	Description string `json:"description,omitempty"`
}

// Component is the minimum operational contract for something that can be
// registered with Opskit.
//
// Component status is descriptive. Status should be fast and should normally
// return cached or local operational state. It should not perform dependency
// checks, reload configuration, call external services, run expensive work,
// dispatch commands, or mutate lifecycle state.
//
// Expensive or active operations belong in Checker, CheckGroup, CommandHandler,
// Workerkit loops, or application-owned execution paths. Admission control
// should use ReadinessContributor when a component needs readiness semantics
// that differ from Status.Ready.
type Component interface {
	ComponentInfo() ComponentInfo
	Status(context.Context) Status
}

// ReadinessPolicy describes how a registered component participates in
// readiness.
type ReadinessPolicy string

const (
	// ReadinessRequired means the component appears in readiness details and
	// blocks aggregate readiness when it is not ready. This is the default
	// registration policy.
	ReadinessRequired ReadinessPolicy = "required"

	// ReadinessOptional means the component appears in readiness details, but
	// does not block aggregate readiness when it is not ready.
	ReadinessOptional ReadinessPolicy = "optional"

	// ReadinessInformational means the component is visible through status and
	// admin snapshots, but is omitted from readiness.
	ReadinessInformational ReadinessPolicy = "informational"
)

// ComponentRegistration describes how a component participates in registry
// readiness views.
type ComponentRegistration struct {
	ReadinessPolicy ReadinessPolicy `json:"readiness_policy"`
}

// ComponentCapabilities describes optional operational capabilities supported by
// a component.
type ComponentCapabilities struct {
	ReadinessContributor bool `json:"readiness_contributor,omitempty"`
	Inspector            bool `json:"inspector,omitempty"`
	Checker              bool `json:"checker,omitempty"`
	CheckDescriber       bool `json:"check_describer,omitempty"`
	CheckGroup           bool `json:"check_group,omitempty"`
	CommandHandler       bool `json:"command_handler,omitempty"`
	CommandDescriber     bool `json:"command_describer,omitempty"`
}

// ComponentSnapshot is the combined operational view of one registered
// component.
type ComponentSnapshot struct {
	Component    ComponentInfo         `json:"component"`
	Registration ComponentRegistration `json:"registration"`
	Capabilities ComponentCapabilities `json:"capabilities"`
	Status       Status                `json:"status"`
	Readiness    *Readiness            `json:"readiness,omitempty"`
	Inspection   *Inspection           `json:"inspection,omitempty"`
	// InspectionError is exposed through operational surfaces when snapshot
	// inspection fails. Inspectors must return only safe, redacted errors.
	InspectionError string `json:"inspection_error,omitempty"`
}

// ComponentFunc is a lightweight Component implementation backed by a function.
//
// It is useful for application-owned operational components that need to report
// status without defining a dedicated struct.
type ComponentFunc struct {
	Info ComponentInfo
	Fn   func(context.Context) Status
}

// ComponentInfo returns the component identity.
func (c ComponentFunc) ComponentInfo() ComponentInfo {
	return c.Info
}

// Status returns the component status.
func (c ComponentFunc) Status(ctx context.Context) Status {
	ctx = normalizeContext(ctx)

	if c.Fn == nil {
		return UnknownStatus("component status function is not configured")
	}

	return c.Fn(ctx)
}
