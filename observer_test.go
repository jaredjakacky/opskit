package opskit

import (
	"context"
	"testing"
	"time"
)

func TestObserverFuncObserve(t *testing.T) {
	ctx := context.Background()
	event := Event{
		Time:      time.Date(2026, 6, 4, 12, 30, 0, 0, time.UTC),
		Operation: "cache/refresh",
	}

	var called bool
	var gotCtx context.Context
	var gotEvent Event

	observer := ObserverFunc(func(ctx context.Context, event Event) {
		called = true
		gotCtx = ctx
		gotEvent = event
	})

	observer.Observe(ctx, event)

	if !called {
		t.Fatal("observer function was not called")
	}
	if gotCtx != ctx {
		t.Fatal("context was not passed through")
	}
	if gotEvent.Time != event.Time {
		t.Fatalf("event.Time = %v, want %v", gotEvent.Time, event.Time)
	}
	if gotEvent.Operation != event.Operation {
		t.Fatalf("event.Operation = %q, want %q", gotEvent.Operation, event.Operation)
	}
}

func TestObserverFuncObserveNormalizesNilContext(t *testing.T) {
	var ctx context.Context

	ObserverFunc(func(ctx context.Context, event Event) {
		if ctx == nil {
			t.Fatal("context is nil, want normalized context")
		}
	}).Observe(ctx, Event{})
}

func TestNilObserverFuncObserveDoesNothing(t *testing.T) {
	var observer ObserverFunc

	observer.Observe(context.Background(), Event{
		Operation: "cache/refresh",
	})
}

func TestNopObserverObserveDoesNothing(t *testing.T) {
	var ctx context.Context

	NopObserver{}.Observe(ctx, Event{
		Operation: "cache/refresh",
	})
}

func TestEventJSONOmitEmptyFields(t *testing.T) {
	event := Event{
		Time: time.Date(2026, 6, 4, 12, 30, 0, 0, time.UTC),
		Component: ComponentInfo{
			Name: "cache",
			Kind: "dependency",
		},
		Operation: "cache/refresh",
		Outcome:   "completed",
		Message:   "refreshed cache",
		Error:     "none",
		Duration:  NewDuration(150 * time.Millisecond),
		Attributes: []Attribute{
			Attr("shard", "primary"),
		},
	}

	requireJSON(t, event, `{"time":"2026-06-04T12:30:00Z","component":{"name":"cache","kind":"dependency"},"operation":"cache/refresh","outcome":"completed","message":"refreshed cache","error":"none","duration":"150ms","attributes":[{"key":"shard","value":"primary"}]}`)
}

func TestEventJSONOmitsOptionalFields(t *testing.T) {
	event := Event{
		Time:      time.Date(2026, 6, 4, 12, 30, 0, 0, time.UTC),
		Operation: "cache/refresh",
	}

	requireJSON(t, event, `{"time":"2026-06-04T12:30:00Z","component":{"name":""},"operation":"cache/refresh"}`)
}
