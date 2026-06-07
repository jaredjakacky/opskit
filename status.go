package opskit

import "time"

// Status is the current operational summary for a component.
type Status struct {
	State      State       `json:"state"`
	Ready      bool        `json:"ready"`
	Message    string      `json:"message,omitempty"`
	UpdatedAt  *time.Time  `json:"updated_at,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
}

// ComponentStatus is the status of one registered component.
type ComponentStatus struct {
	Component    ComponentInfo         `json:"component"`
	Registration ComponentRegistration `json:"registration"`
	Capabilities ComponentCapabilities `json:"capabilities"`
	Status       Status                `json:"status"`
}

// SystemStatus is the aggregate status view of registered components.
//
// SystemStatus deliberately does not derive aggregate readiness or a single
// aggregate state. Use Registry.Readiness for readiness. Use the component
// entries here for operational status.
type SystemStatus struct {
	Components []ComponentStatus `json:"components,omitempty"`
}

// ReadyStatus returns a ready component status.
func ReadyStatus(message string, attrs ...Attribute) Status {
	return Status{
		State:      StateReady,
		Ready:      true,
		Message:    message,
		UpdatedAt:  nowUTC(),
		Attributes: cloneAttributes(attrs),
	}
}

// DegradedStatus returns a degraded but still ready component status.
func DegradedStatus(message string, attrs ...Attribute) Status {
	return Status{
		State:      StateDegraded,
		Ready:      true,
		Message:    message,
		UpdatedAt:  nowUTC(),
		Attributes: cloneAttributes(attrs),
	}
}

// NotReadyStatus returns a not-ready component status.
func NotReadyStatus(message string, attrs ...Attribute) Status {
	return Status{
		State:      StateNotReady,
		Ready:      false,
		Message:    message,
		UpdatedAt:  nowUTC(),
		Attributes: cloneAttributes(attrs),
	}
}

// FailedStatus returns a failed component status.
func FailedStatus(message string, attrs ...Attribute) Status {
	return Status{
		State:      StateFailed,
		Ready:      false,
		Message:    message,
		UpdatedAt:  nowUTC(),
		Attributes: cloneAttributes(attrs),
	}
}

// UnknownStatus returns an unknown component status.
func UnknownStatus(message string, attrs ...Attribute) Status {
	return Status{
		State:      StateUnknown,
		Ready:      false,
		Message:    message,
		UpdatedAt:  nowUTC(),
		Attributes: cloneAttributes(attrs),
	}
}
