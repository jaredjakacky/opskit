package opskit

import (
	"context"
	"time"
)

// Event describes one operational event emitted by a component.
//
// Event is intentionally small and backend-neutral. Kits can map Events to
// slog records, OpenTelemetry spans/metrics, test collectors, or custom sinks.
// The root package does not import or configure telemetry backends.
type Event struct {
	Time       time.Time     `json:"time"`
	Component  ComponentInfo `json:"component"`
	Operation  string        `json:"operation"`
	Outcome    string        `json:"outcome,omitempty"`
	Message    string        `json:"message,omitempty"`
	Error      string        `json:"error,omitempty"`
	Duration   Duration      `json:"duration,omitempty"`
	Attributes []Attribute   `json:"attributes,omitempty"`
}

// Observer receives operational events.
type Observer interface {
	Observe(context.Context, Event)
}

// ObserverFunc adapts a function into an Observer.
type ObserverFunc func(context.Context, Event)

// Observe passes an event to the function observer when the caller explicitly
// calls it.
func (fn ObserverFunc) Observe(ctx context.Context, event Event) {
	ctx = normalizeContext(ctx)

	if fn != nil {
		fn(ctx, event)
	}
}

// NopObserver is an Observer implementation that ignores observed events.
type NopObserver struct{}

// Observe ignores the event.
func (NopObserver) Observe(context.Context, Event) {}
