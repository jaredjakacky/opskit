package opskit

import (
	"context"
	"testing"
	"time"
)

func TestNormalizeContextReturnsBackgroundForNilContext(t *testing.T) {
	var ctx context.Context

	got := normalizeContext(ctx)
	if got == nil {
		t.Fatal("normalizeContext returned nil, want background context")
	}
	if err := got.Err(); err != nil {
		t.Fatalf("normalized context Err() = %v, want nil", err)
	}
}

func TestNormalizeContextPreservesNonNilContext(t *testing.T) {
	type contextKey string

	ctx := context.WithValue(context.Background(), contextKey("key"), "value")

	got := normalizeContext(ctx)
	if got != ctx {
		t.Fatal("normalizeContext did not preserve non-nil context")
	}
	if got.Value(contextKey("key")) != "value" {
		t.Fatalf("context value = %v, want value", got.Value(contextKey("key")))
	}
}

func TestRegistryMethodsNormalizeNilContext(t *testing.T) {
	registry := NewRegistry()
	var ctx context.Context
	component := contextCheckingComponent{
		t:    t,
		info: ComponentInfo{Name: "component", Kind: "test"},
	}

	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	status := registry.Status(ctx)
	if len(status.Components) != 1 {
		t.Fatalf("Status.Components length = %d, want 1", len(status.Components))
	}

	readiness := registry.Readiness(ctx)
	if !readiness.Ready {
		t.Fatal("Readiness.Ready = false, want true")
	}

	snapshot, err := registry.Snapshot(ctx, "component")
	if err != nil {
		t.Fatalf("Snapshot error = %v", err)
	}
	if snapshot.Inspection == nil {
		t.Fatal("Snapshot.Inspection is nil, want inspection")
	}

	if _, err := registry.Inspect(ctx, "component"); err != nil {
		t.Fatalf("Inspect error = %v", err)
	}
}

func TestFunctionAdaptersNormalizeNilContext(t *testing.T) {
	var ctx context.Context

	CheckFunc(func(ctx context.Context) CheckResult {
		requireContext(t, ctx)
		return ReadyCheck("ready", 0)
	}).Check(ctx)

	CheckGroupFunc(func(ctx context.Context) CheckSummary {
		requireContext(t, ctx)
		return SummarizeChecks("ready", time.Now().UTC(), nil)
	}).CheckAll(ctx)

	CommandHandlerFunc(func(ctx context.Context, request CommandRequest) CommandResult {
		requireContext(t, ctx)
		return CompletedCommand("completed", nil, 0)
	}).HandleCommand(ctx, CommandRequest{Name: "test"})

	ComponentFunc{
		Info: ComponentInfo{Name: "component", Kind: "test"},
		Fn: func(ctx context.Context) Status {
			requireContext(t, ctx)
			return ReadyStatus("ready")
		},
	}.Status(ctx)
}

type contextCheckingComponent struct {
	t    *testing.T
	info ComponentInfo
}

func (c contextCheckingComponent) ComponentInfo() ComponentInfo {
	return c.info
}

func (c contextCheckingComponent) Status(ctx context.Context) Status {
	requireContext(c.t, ctx)
	return ReadyStatus("ready")
}

func (c contextCheckingComponent) Readiness(ctx context.Context) Readiness {
	requireContext(c.t, ctx)
	return ReadyReadiness("ready")
}

func (c contextCheckingComponent) Inspect(ctx context.Context) (Inspection, error) {
	requireContext(c.t, ctx)
	return Inspection{Summary: "ok"}, nil
}

func requireContext(t *testing.T, ctx context.Context) {
	t.Helper()

	if ctx == nil {
		t.Fatal("context is nil, want normalized context")
	}
}
