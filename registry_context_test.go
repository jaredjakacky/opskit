package opskit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRegistryContextCanceledBeforeInvocationSkipsComponentCallbacks(t *testing.T) {
	component := &registryContextComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
	}
	registry := NewRegistry()
	registry.MustRegister(component)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	requireCanceledSystemStatus(t, registry.Status(ctx), 0, context.Canceled)
	requireCanceledRegistryReadiness(t, registry.Readiness(ctx), 0, context.Canceled)
	if _, err := registry.Snapshot(ctx, "component"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Snapshot error = %v, want context canceled", err)
	}
	if _, err := registry.Inspect(ctx, "component"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Inspect error = %v, want context canceled", err)
	}
	if _, err := registry.Checks(ctx, "component"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Checks error = %v, want context canceled", err)
	}
	if _, err := registry.Commands(ctx, "component"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Commands error = %v, want context canceled", err)
	}

	if component.statusCalls != 0 || component.readinessCalls != 0 || component.inspectCalls != 0 || component.checksCalls != 0 || component.commandsCalls != 0 {
		t.Fatalf("callback counts = status:%d readiness:%d inspect:%d checks:%d commands:%d, want all zero",
			component.statusCalls,
			component.readinessCalls,
			component.inspectCalls,
			component.checksCalls,
			component.commandsCalls,
		)
	}
}

func TestRegistryStatusReportsCancellationForEmptyRegistry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	status := NewRegistry().Status(ctx)
	requireCanceledSystemStatus(t, status, 0, context.Canceled)
}

func TestRegistryStatusRejectsResultWhenContextCanceledDuringCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	component := &registryContextComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
		statusFn: func(context.Context) Status {
			cancel()
			return ReadyStatus("late ready")
		},
	}
	later := &registryContextComponent{
		info: ComponentInfo{Name: "later", Kind: "test"},
	}
	registry := NewRegistry()
	registry.MustRegister(component)
	registry.MustRegister(later)

	status := registry.Status(ctx)
	requireCanceledSystemStatus(t, status, 0, context.Canceled)
	if component.statusCalls != 1 {
		t.Fatalf("status calls = %d, want 1", component.statusCalls)
	}
	if later.statusCalls != 0 {
		t.Fatalf("later status calls = %d, want 0", later.statusCalls)
	}
}

func TestRegistryStatusReportsCancellationAtFinalAcceptance(t *testing.T) {
	ctx := newRegistryCheckpointContext(4)
	component := &registryContextComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
	}
	registry := NewRegistry()
	registry.MustRegister(component)

	status := registry.Status(ctx)
	requireCanceledSystemStatus(t, status, 1, context.DeadlineExceeded)
	if status.Components[0].Component.Name != "component" {
		t.Fatalf("first component = %q, want accepted component", status.Components[0].Component.Name)
	}
}

func TestRegistryReadinessReportsCancellationForEmptyRegistry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	readiness := NewRegistry().Readiness(ctx)
	requireCanceledRegistryReadiness(t, readiness, 0, context.Canceled)
}

func TestRegistryReadinessRejectsContributorResultWhenContextCanceledDuringCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	component := &registryContextComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
		readinessFn: func(context.Context) Readiness {
			cancel()
			return Readiness{Ready: true, Reason: "late ready"}
		},
	}
	later := &registryContextComponent{
		info: ComponentInfo{Name: "later", Kind: "test"},
	}
	registry := NewRegistry()
	registry.MustRegister(component)
	registry.MustRegister(later)

	readiness := registry.Readiness(ctx)
	requireCanceledRegistryReadiness(t, readiness, 0, context.Canceled)
	if component.readinessCalls != 1 {
		t.Fatalf("readiness calls = %d, want 1", component.readinessCalls)
	}
	if later.readinessCalls != 0 {
		t.Fatalf("later readiness calls = %d, want 0", later.readinessCalls)
	}
}

func TestRegistryReadinessRejectsStatusFallbackWhenContextCanceledDuringCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	component := &registryContextStatusOnlyComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
		statusFn: func(context.Context) Status {
			cancel()
			return ReadyStatus("late ready")
		},
	}
	registry := NewRegistry()
	registry.MustRegister(component)

	readiness := registry.Readiness(ctx)
	requireCanceledRegistryReadiness(t, readiness, 0, context.Canceled)
	if component.statusCalls != 1 {
		t.Fatalf("status calls = %d, want 1", component.statusCalls)
	}
}

func TestRegistryReadinessReportsCancellationAtFinalAcceptance(t *testing.T) {
	ctx := newRegistryCheckpointContext(4)
	component := &registryContextStatusOnlyComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
	}
	registry := NewRegistry()
	registry.MustRegister(component)

	readiness := registry.Readiness(ctx)
	requireCanceledRegistryReadiness(t, readiness, 1, context.DeadlineExceeded)
	if readiness.Components[0].Component.Name != "component" {
		t.Fatalf("first readiness component = %q, want accepted component", readiness.Components[0].Component.Name)
	}
}

func TestRegistrySnapshotRejectsContextExpirationDuringEachCallback(t *testing.T) {
	t.Run("status", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		component := &registryContextComponent{
			info: ComponentInfo{Name: "component", Kind: "test"},
			statusFn: func(context.Context) Status {
				cancel()
				return ReadyStatus("late ready")
			},
		}
		registry := NewRegistry()
		registry.MustRegister(component)

		snapshot, err := registry.Snapshot(ctx, "component")
		requireZeroSnapshotContextError(t, snapshot, err, context.Canceled)
		if component.readinessCalls != 0 || component.inspectCalls != 0 {
			t.Fatalf("later callback counts = readiness:%d inspect:%d, want zero", component.readinessCalls, component.inspectCalls)
		}
	})

	t.Run("readiness", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		component := &registryContextComponent{
			info: ComponentInfo{Name: "component", Kind: "test"},
			readinessFn: func(context.Context) Readiness {
				cancel()
				return Readiness{Ready: true, Reason: "late ready"}
			},
		}
		registry := NewRegistry()
		registry.MustRegister(component)

		snapshot, err := registry.Snapshot(ctx, "component")
		requireZeroSnapshotContextError(t, snapshot, err, context.Canceled)
		if component.inspectCalls != 0 {
			t.Fatalf("inspect calls = %d, want zero", component.inspectCalls)
		}
	})

	t.Run("inspection success", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		component := &registryContextComponent{
			info: ComponentInfo{Name: "component", Kind: "test"},
			inspectFn: func(context.Context) (Inspection, error) {
				cancel()
				return Inspection{Summary: "late inspection"}, nil
			},
		}
		registry := NewRegistry()
		registry.MustRegister(component)

		snapshot, err := registry.Snapshot(ctx, "component")
		requireZeroSnapshotContextError(t, snapshot, err, context.Canceled)
	})

	t.Run("inspection error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		component := &registryContextComponent{
			info: ComponentInfo{Name: "component", Kind: "test"},
			inspectFn: func(context.Context) (Inspection, error) {
				cancel()
				return Inspection{}, errors.New("inspection failed")
			},
		}
		registry := NewRegistry()
		registry.MustRegister(component)

		snapshot, err := registry.Snapshot(ctx, "component")
		requireZeroSnapshotContextError(t, snapshot, err, context.Canceled)
	})

	t.Run("inspection panic", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		component := &registryContextComponent{
			info: ComponentInfo{Name: "component", Kind: "test"},
			inspectFn: func(context.Context) (Inspection, error) {
				cancel()
				panic("inspection panic")
			},
		}
		registry := NewRegistry()
		registry.MustRegister(component)

		snapshot, err := registry.Snapshot(ctx, "component")
		requireZeroSnapshotContextError(t, snapshot, err, context.Canceled)
	})
}

func TestRegistrySnapshotDeadlineDuringInspectorReturnsDeadlineError(t *testing.T) {
	component := &registryContextComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
		inspectFn: func(ctx context.Context) (Inspection, error) {
			<-ctx.Done()
			return Inspection{}, ctx.Err()
		},
	}
	registry := NewRegistry()
	registry.MustRegister(component)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	snapshot, err := registry.Snapshot(ctx, "component")
	requireZeroSnapshotContextError(t, snapshot, err, context.DeadlineExceeded)
}

func TestRegistrySnapshotTreatsInspectorDeadlineErrorAsFailureWhileParentContextIsActive(t *testing.T) {
	component := &registryContextComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
		inspectFn: func(context.Context) (Inspection, error) {
			return Inspection{}, context.DeadlineExceeded
		},
	}
	registry := NewRegistry()
	registry.MustRegister(component)

	snapshot, err := registry.Snapshot(context.Background(), "component")
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	if snapshot.InspectionFailure == nil || snapshot.InspectionFailure.Code != FailureCodeInspectionFailed {
		t.Fatalf("inspection failure = %#v, want %q", snapshot.InspectionFailure, FailureCodeInspectionFailed)
	}
}

func TestRegistrySnapshotRejectsContextExpirationAtFinalAcceptance(t *testing.T) {
	ctx := newRegistryCheckpointContext(4)
	component := &registryContextStatusOnlyComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
	}
	registry := NewRegistry()
	registry.MustRegister(component)

	snapshot, err := registry.Snapshot(ctx, "component")
	requireZeroSnapshotContextError(t, snapshot, err, context.DeadlineExceeded)
}

func TestRegistryInspectContextCancellationOverridesCallbackOutcome(t *testing.T) {
	tests := []struct {
		name      string
		inspectFn func(context.Context, context.CancelFunc) (Inspection, error)
	}{
		{
			name: "success",
			inspectFn: func(_ context.Context, cancel context.CancelFunc) (Inspection, error) {
				cancel()
				return Inspection{Summary: "late inspection"}, nil
			},
		},
		{
			name: "error",
			inspectFn: func(_ context.Context, cancel context.CancelFunc) (Inspection, error) {
				cancel()
				return Inspection{}, errors.New("inspection failed")
			},
		},
		{
			name: "panic",
			inspectFn: func(_ context.Context, cancel context.CancelFunc) (Inspection, error) {
				cancel()
				panic("inspection panic")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			component := &registryContextComponent{
				info: ComponentInfo{Name: "component", Kind: "test"},
				inspectFn: func(callbackCtx context.Context) (Inspection, error) {
					return tt.inspectFn(callbackCtx, cancel)
				},
			}
			registry := NewRegistry()
			registry.MustRegister(component)

			inspection, err := registry.Inspect(ctx, "component")
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Inspect error = %v, want context canceled", err)
			}
			if inspection.Summary != nil || inspection.Details != nil || len(inspection.Attributes) != 0 {
				t.Fatalf("inspection = %#v, want zero value", inspection)
			}
		})
	}
}

func TestRegistryChecksContextCancellationOverridesCallbackOutcome(t *testing.T) {
	tests := []struct {
		name     string
		checksFn func(context.CancelFunc) []CheckDescriptor
	}{
		{
			name: "result",
			checksFn: func(cancel context.CancelFunc) []CheckDescriptor {
				cancel()
				return []CheckDescriptor{{Name: "late"}}
			},
		},
		{
			name: "panic",
			checksFn: func(cancel context.CancelFunc) []CheckDescriptor {
				cancel()
				panic("checks panic")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			component := &registryContextComponent{
				info: ComponentInfo{Name: "component", Kind: "test"},
				checksFn: func(context.Context) []CheckDescriptor {
					return tt.checksFn(cancel)
				},
			}
			registry := NewRegistry()
			registry.MustRegister(component)

			checks, err := registry.Checks(ctx, "component")
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Checks error = %v, want context canceled", err)
			}
			if checks != nil {
				t.Fatalf("checks = %#v, want nil", checks)
			}
		})
	}
}

func TestRegistryChecksRejectsContextExpirationDuringClone(t *testing.T) {
	ctx := newRegistryCheckpointContext(3)
	component := &registryContextComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
	}
	registry := NewRegistry()
	registry.MustRegister(component)

	checks, err := registry.Checks(ctx, "component")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Checks error = %v, want deadline exceeded", err)
	}
	if checks != nil {
		t.Fatalf("checks = %#v, want nil", checks)
	}
}

func TestRegistryCommandsContextCancellationOverridesCallbackOutcome(t *testing.T) {
	tests := []struct {
		name       string
		commandsFn func(context.CancelFunc) []CommandDescriptor
	}{
		{
			name: "result",
			commandsFn: func(cancel context.CancelFunc) []CommandDescriptor {
				cancel()
				return []CommandDescriptor{{Name: "late/run"}}
			},
		},
		{
			name: "panic",
			commandsFn: func(cancel context.CancelFunc) []CommandDescriptor {
				cancel()
				panic("commands panic")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			component := &registryContextComponent{
				info: ComponentInfo{Name: "component", Kind: "test"},
				commandsFn: func(context.Context) []CommandDescriptor {
					return tt.commandsFn(cancel)
				},
			}
			registry := NewRegistry()
			registry.MustRegister(component)

			commands, err := registry.Commands(ctx, "component")
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Commands error = %v, want context canceled", err)
			}
			if commands != nil {
				t.Fatalf("commands = %#v, want nil", commands)
			}
		})
	}
}

func TestRegistryCommandsRejectsContextExpirationDuringClone(t *testing.T) {
	ctx := newRegistryCheckpointContext(3)
	component := &registryContextComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
	}
	registry := NewRegistry()
	registry.MustRegister(component)

	commands, err := registry.Commands(ctx, "component")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Commands error = %v, want deadline exceeded", err)
	}
	if commands != nil {
		t.Fatalf("commands = %#v, want nil", commands)
	}
}

func requireCanceledSystemStatus(t *testing.T, status SystemStatus, accepted int, wantErr error) {
	t.Helper()

	if len(status.Components) != accepted+1 {
		t.Fatalf("status component count = %d, want %d accepted plus cancellation", len(status.Components), accepted)
	}
	canceled := status.Components[len(status.Components)-1]
	if canceled.Component.Name != "opskit.registry" || canceled.Status.State != StateUnknown || canceled.Status.Ready {
		t.Fatalf("cancellation status = %#v, want unknown not-ready opskit.registry", canceled)
	}
	if len(canceled.Status.Attributes) != 1 || canceled.Status.Attributes[0] != Attr("error", contextFailureMessage(wantErr)) {
		t.Fatalf("cancellation attributes = %#v, want classified context error", canceled.Status.Attributes)
	}
}

func requireCanceledRegistryReadiness(t *testing.T, readiness SystemReadiness, accepted int, wantErr error) {
	t.Helper()

	if readiness.Ready || readiness.Reason != "readiness evaluation canceled" {
		t.Fatalf("readiness = %#v, want canceled and not ready", readiness)
	}
	if len(readiness.Components) != accepted+1 {
		t.Fatalf("readiness component count = %d, want %d accepted plus cancellation", len(readiness.Components), accepted)
	}
	canceled := readiness.Components[len(readiness.Components)-1]
	if canceled.Component.Name != "opskit.registry" || canceled.Registration.ReadinessPolicy != ReadinessRequired || canceled.Readiness.Ready || canceled.Readiness.Reason != "readiness evaluation canceled" {
		t.Fatalf("cancellation component = %#v, want required not-ready opskit.registry", canceled)
	}
	if len(canceled.Readiness.Items) != 1 {
		t.Fatalf("cancellation item count = %d, want 1", len(canceled.Readiness.Items))
	}
	item := canceled.Readiness.Items[0]
	if item.Name != "opskit.registry" || item.State != StateUnknown || item.Ready || item.Message != contextFailureMessage(wantErr) {
		t.Fatalf("cancellation item = %#v, want classified context cancellation", item)
	}
}

func requireZeroSnapshotContextError(t *testing.T, snapshot ComponentSnapshot, err, want error) {
	t.Helper()

	if !errors.Is(err, want) {
		t.Fatalf("Snapshot error = %v, want %v", err, want)
	}
	if snapshot.Component.Name != "" || snapshot.Status.State != "" || snapshot.Readiness != nil || snapshot.Inspection != nil || snapshot.InspectionFailure != nil {
		t.Fatalf("snapshot = %#v, want zero value", snapshot)
	}
}

type registryCheckpointContext struct {
	calls  int
	failAt int
	done   chan struct{}
	failed bool
}

func newRegistryCheckpointContext(failAt int) *registryCheckpointContext {
	return &registryCheckpointContext{
		failAt: failAt,
		done:   make(chan struct{}),
	}
}

func (*registryCheckpointContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (c *registryCheckpointContext) Done() <-chan struct{} {
	return c.done
}

func (c *registryCheckpointContext) Err() error {
	c.calls++
	if c.calls >= c.failAt {
		if !c.failed {
			close(c.done)
			c.failed = true
		}
		return context.DeadlineExceeded
	}
	return nil
}

func (*registryCheckpointContext) Value(any) any {
	return nil
}

type registryContextStatusOnlyComponent struct {
	info        ComponentInfo
	statusFn    func(context.Context) Status
	statusCalls int
}

func (c *registryContextStatusOnlyComponent) ComponentInfo() ComponentInfo {
	return c.info
}

func (c *registryContextStatusOnlyComponent) Status(ctx context.Context) Status {
	c.statusCalls++
	if c.statusFn != nil {
		return c.statusFn(ctx)
	}
	return ReadyStatus("ready")
}

type registryContextComponent struct {
	info ComponentInfo

	statusFn    func(context.Context) Status
	readinessFn func(context.Context) Readiness
	inspectFn   func(context.Context) (Inspection, error)
	checksFn    func(context.Context) []CheckDescriptor
	commandsFn  func(context.Context) []CommandDescriptor

	statusCalls    int
	readinessCalls int
	inspectCalls   int
	checksCalls    int
	commandsCalls  int
}

func (c *registryContextComponent) ComponentInfo() ComponentInfo {
	return c.info
}

func (c *registryContextComponent) Status(ctx context.Context) Status {
	c.statusCalls++
	if c.statusFn != nil {
		return c.statusFn(ctx)
	}
	return ReadyStatus("ready")
}

func (c *registryContextComponent) Readiness(ctx context.Context) Readiness {
	c.readinessCalls++
	if c.readinessFn != nil {
		return c.readinessFn(ctx)
	}
	return Readiness{Ready: true, Reason: "ready"}
}

func (c *registryContextComponent) Inspect(ctx context.Context) (Inspection, error) {
	c.inspectCalls++
	if c.inspectFn != nil {
		return c.inspectFn(ctx)
	}
	return Inspection{Summary: "ok"}, nil
}

func (c *registryContextComponent) Checks(ctx context.Context) []CheckDescriptor {
	c.checksCalls++
	if c.checksFn != nil {
		return c.checksFn(ctx)
	}
	return []CheckDescriptor{{Name: "component/check", Attributes: []Attribute{Attr("scope", "test")}}}
}

func (c *registryContextComponent) Commands(ctx context.Context) []CommandDescriptor {
	c.commandsCalls++
	if c.commandsFn != nil {
		return c.commandsFn(ctx)
	}
	return []CommandDescriptor{{Name: "component/run", Attributes: []Attribute{Attr("scope", "test")}}}
}
