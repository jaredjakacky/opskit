package opskit

import (
	"context"
	"errors"
	"testing"
)

func TestRegistryRegisterAndLookup(t *testing.T) {
	var registry Registry

	first := testComponent{
		info:   ComponentInfo{Name: "first", Kind: "test"},
		status: ReadyStatus("first ready"),
	}
	second := testComponent{
		info:   ComponentInfo{Name: "second", Kind: "test"},
		status: ReadyStatus("second ready"),
	}

	if err := registry.Register(first, nil); err != nil {
		t.Fatalf("Register(first) error = %v", err)
	}
	if err := registry.Register(second, Optional()); err != nil {
		t.Fatalf("Register(second) error = %v", err)
	}

	component, ok := registry.Component("first")
	if !ok {
		t.Fatal("Component(first) ok = false, want true")
	}
	if got := component.ComponentInfo().Name; got != "first" {
		t.Fatalf("Component(first).Name = %q, want first", got)
	}

	if _, ok := registry.Component("missing"); ok {
		t.Fatal("Component(missing) ok = true, want false")
	}

	components := registry.Components()
	if len(components) != 2 {
		t.Fatalf("Components length = %d, want 2", len(components))
	}
	if got := components[0].ComponentInfo().Name; got != "first" {
		t.Fatalf("Components[0].Name = %q, want first", got)
	}
	if got := components[1].ComponentInfo().Name; got != "second" {
		t.Fatalf("Components[1].Name = %q, want second", got)
	}

	components[0] = second
	components = registry.Components()
	if got := components[0].ComponentInfo().Name; got != "first" {
		t.Fatalf("Components returned mutable registry slice, first name = %q", got)
	}
}

func TestRegistryRegisterRejectsNilAndDuplicateComponents(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Register(nil); err != ErrNilComponent {
		t.Fatalf("Register(nil) error = %v, want %v", err, ErrNilComponent)
	}

	component := testComponent{
		info:   ComponentInfo{Name: "component", Kind: "test"},
		status: ReadyStatus("ready"),
	}

	if err := registry.Register(component); err != nil {
		t.Fatalf("Register(component) error = %v", err)
	}
	if err := registry.Register(component); err != ErrDuplicateComponent {
		t.Fatalf("Register(duplicate) error = %v, want %v", err, ErrDuplicateComponent)
	}
}

func TestRegistryMustRegisterPanicsOnError(t *testing.T) {
	registry := NewRegistry()

	defer func() {
		if recover() == nil {
			t.Fatal("MustRegister did not panic")
		}
	}()

	registry.MustRegister(nil)
}

func TestRegistryStatusCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	component := &countingComponent{
		info:   ComponentInfo{Name: "component", Kind: "test"},
		status: ReadyStatus("ready"),
	}
	registry := NewRegistry()
	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	status := registry.Status(ctx)
	if component.statusCalls != 0 {
		t.Fatalf("status calls = %d, want 0", component.statusCalls)
	}
	if len(status.Components) != 1 {
		t.Fatalf("Status.Components length = %d, want 1", len(status.Components))
	}

	got := status.Components[0]
	if got.Component.Name != "opskit.registry" {
		t.Fatalf("Component.Name = %q, want opskit.registry", got.Component.Name)
	}
	if got.Status.State != StateUnknown {
		t.Fatalf("Status.State = %q, want %q", got.Status.State, StateUnknown)
	}
	if got.Status.Ready {
		t.Fatal("Status.Ready = true, want false")
	}
	if len(got.Status.Attributes) != 1 || got.Status.Attributes[0].Key != "error" {
		t.Fatalf("Status.Attributes = %+v, want one error attribute", got.Status.Attributes)
	}
}

func TestRegistryReadinessCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	component := &countingComponent{
		info:   ComponentInfo{Name: "component", Kind: "test"},
		status: ReadyStatus("ready"),
	}
	registry := NewRegistry()
	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	readiness := registry.Readiness(ctx)
	if component.statusCalls != 0 {
		t.Fatalf("status calls = %d, want 0", component.statusCalls)
	}
	if readiness.Ready {
		t.Fatal("Readiness.Ready = true, want false")
	}
	if readiness.Reason != "readiness evaluation canceled" {
		t.Fatalf("Readiness.Reason = %q, want readiness evaluation canceled", readiness.Reason)
	}
	if len(readiness.Components) != 1 {
		t.Fatalf("Readiness.Components length = %d, want 1", len(readiness.Components))
	}
	if got := readiness.Components[0].Name; got != "opskit.registry" {
		t.Fatalf("Readiness.Components[0].Name = %q, want opskit.registry", got)
	}
}

func TestRegistryReadinessPolicies(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	required := &testReadinessComponent{
		info:      ComponentInfo{Name: "required", Kind: "test"},
		status:    ReadyStatus("required ready"),
		readiness: ReadyReadiness("required ready"),
	}
	optional := testComponent{
		info:   ComponentInfo{Name: "optional", Kind: "test"},
		status: NotReadyStatus("optional not ready"),
	}
	informational := &testReadinessComponent{
		info:      ComponentInfo{Name: "informational", Kind: "test"},
		status:    NotReadyStatus("informational not ready"),
		readiness: NotReadyReadiness("informational readiness"),
	}

	if err := registry.Register(required); err != nil {
		t.Fatalf("Register(required) error = %v", err)
	}
	if err := registry.Register(optional, Optional()); err != nil {
		t.Fatalf("Register(optional) error = %v", err)
	}
	if err := registry.Register(informational, Informational()); err != nil {
		t.Fatalf("Register(informational) error = %v", err)
	}

	readiness := registry.Readiness(ctx)
	if !readiness.Ready {
		t.Fatalf("Readiness.Ready = false, want true")
	}
	if len(readiness.Components) != 2 {
		t.Fatalf("Readiness.Components length = %d, want 2", len(readiness.Components))
	}
	if readiness.Components[0].Name != "required" {
		t.Fatalf("Readiness.Components[0].Name = %q, want required", readiness.Components[0].Name)
	}
	if readiness.Components[0].Policy != ReadinessRequired {
		t.Fatalf("Readiness.Components[0].Policy = %q, want %q", readiness.Components[0].Policy, ReadinessRequired)
	}
	if readiness.Components[1].Name != "optional" {
		t.Fatalf("Readiness.Components[1].Name = %q, want optional", readiness.Components[1].Name)
	}
	if readiness.Components[1].Policy != ReadinessOptional {
		t.Fatalf("Readiness.Components[1].Policy = %q, want %q", readiness.Components[1].Policy, ReadinessOptional)
	}
	if readiness.Components[1].Ready {
		t.Fatal("Readiness.Components[1].Ready = true, want false")
	}

	status := registry.Status(ctx)
	if len(status.Components) != 3 {
		t.Fatalf("Status.Components length = %d, want 3", len(status.Components))
	}

	wantPolicies := map[string]ReadinessPolicy{
		"required":      ReadinessRequired,
		"optional":      ReadinessOptional,
		"informational": ReadinessInformational,
	}
	for _, component := range status.Components {
		wantPolicy := wantPolicies[component.Component.Name]
		if component.Registration.ReadinessPolicy != wantPolicy {
			t.Fatalf("%s readiness policy = %q, want %q", component.Component.Name, component.Registration.ReadinessPolicy, wantPolicy)
		}
	}
}

func TestRegistryReadinessSetsPolicyOnContributorItems(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := &testReadinessComponent{
		info:   ComponentInfo{Name: "component", Kind: "test"},
		status: ReadyStatus("ready"),
		readiness: ReadyReadiness("ready", ReadinessItem{
			Name:  "dependency",
			Ready: false,
			State: StateNotReady,
		}),
	}

	if err := registry.Register(component, Optional()); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	readiness := registry.Readiness(ctx)
	if len(readiness.Components) != 1 {
		t.Fatalf("Readiness.Components length = %d, want 1", len(readiness.Components))
	}
	if readiness.Components[0].Name != "dependency" {
		t.Fatalf("Readiness.Components[0].Name = %q, want dependency", readiness.Components[0].Name)
	}
	if readiness.Components[0].Kind != "test" {
		t.Fatalf("Readiness.Components[0].Kind = %q, want test", readiness.Components[0].Kind)
	}
	if readiness.Components[0].Policy != ReadinessOptional {
		t.Fatalf("Readiness.Components[0].Policy = %q, want %q", readiness.Components[0].Policy, ReadinessOptional)
	}
}

func TestRegistrySnapshotReadinessPolicies(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	optional := testComponent{
		info:   ComponentInfo{Name: "optional", Kind: "test"},
		status: NotReadyStatus("optional not ready"),
	}
	required := &testReadinessComponent{
		info:   ComponentInfo{Name: "required", Kind: "test"},
		status: ReadyStatus("required ready"),
		readiness: ReadyReadiness("required ready", ReadinessItem{
			Name:  "required-dependency",
			Ready: true,
			State: StateReady,
		}),
	}
	informational := &testReadinessComponent{
		info:      ComponentInfo{Name: "informational", Kind: "test"},
		status:    ReadyStatus("informational ready"),
		readiness: NotReadyReadiness("informational readiness"),
	}

	if err := registry.Register(optional, Optional()); err != nil {
		t.Fatalf("Register(optional) error = %v", err)
	}
	if err := registry.Register(required); err != nil {
		t.Fatalf("Register(required) error = %v", err)
	}
	if err := registry.Register(informational, Informational()); err != nil {
		t.Fatalf("Register(informational) error = %v", err)
	}

	optionalSnapshot, err := registry.Snapshot(ctx, "optional")
	if err != nil {
		t.Fatalf("Snapshot(optional) error = %v", err)
	}
	if optionalSnapshot.Readiness == nil {
		t.Fatal("Snapshot(optional).Readiness is nil, want derived readiness")
	}
	if optionalSnapshot.Readiness.Ready {
		t.Fatal("Snapshot(optional).Readiness.Ready = true, want false")
	}
	if len(optionalSnapshot.Readiness.Components) != 1 {
		t.Fatalf("Snapshot(optional).Readiness.Components length = %d, want 1", len(optionalSnapshot.Readiness.Components))
	}
	if optionalSnapshot.Readiness.Components[0].Policy != ReadinessOptional {
		t.Fatalf("Snapshot(optional).Readiness.Components[0].Policy = %q, want %q", optionalSnapshot.Readiness.Components[0].Policy, ReadinessOptional)
	}

	requiredSnapshot, err := registry.Snapshot(ctx, "required")
	if err != nil {
		t.Fatalf("Snapshot(required) error = %v", err)
	}
	if requiredSnapshot.Registration.ReadinessPolicy != ReadinessRequired {
		t.Fatalf("Snapshot(required).Registration.ReadinessPolicy = %q, want %q", requiredSnapshot.Registration.ReadinessPolicy, ReadinessRequired)
	}
	if requiredSnapshot.Readiness == nil {
		t.Fatal("Snapshot(required).Readiness is nil, want contributor readiness")
	}
	if len(requiredSnapshot.Readiness.Components) != 1 {
		t.Fatalf("Snapshot(required).Readiness.Components length = %d, want 1", len(requiredSnapshot.Readiness.Components))
	}
	if requiredSnapshot.Readiness.Components[0].Policy != ReadinessRequired {
		t.Fatalf("Snapshot(required).Readiness.Components[0].Policy = %q, want %q", requiredSnapshot.Readiness.Components[0].Policy, ReadinessRequired)
	}

	informationalSnapshot, err := registry.Snapshot(ctx, "informational")
	if err != nil {
		t.Fatalf("Snapshot(informational) error = %v", err)
	}
	if informationalSnapshot.Readiness != nil {
		t.Fatal("Snapshot(informational).Readiness is not nil, want nil")
	}
	if informational.readinessCalls != 0 {
		t.Fatalf("informational readiness calls = %d, want 0", informational.readinessCalls)
	}
}

func TestRegistrySnapshotIncludesCapabilitiesReadinessAndInspection(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := &testOperationalComponent{
		info:       ComponentInfo{Name: "component", Kind: "test"},
		status:     ReadyStatus("ready"),
		readiness:  ReadyReadiness("ready"),
		inspection: Inspection{Summary: "ok"},
	}

	if err := registry.Register(component, Optional()); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	snapshot, err := registry.Snapshot(ctx, "component")
	if err != nil {
		t.Fatalf("Snapshot error = %v", err)
	}

	if snapshot.Component.Name != "component" {
		t.Fatalf("Snapshot.Component.Name = %q, want component", snapshot.Component.Name)
	}
	if snapshot.Registration.ReadinessPolicy != ReadinessOptional {
		t.Fatalf("Snapshot.ReadinessPolicy = %q, want %q", snapshot.Registration.ReadinessPolicy, ReadinessOptional)
	}
	if !snapshot.Capabilities.ReadinessContributor ||
		!snapshot.Capabilities.Inspector ||
		!snapshot.Capabilities.Checker ||
		!snapshot.Capabilities.CheckDescriber ||
		!snapshot.Capabilities.CheckGroup ||
		!snapshot.Capabilities.CommandHandler ||
		!snapshot.Capabilities.CommandDescriber {
		t.Fatalf("Snapshot.Capabilities = %+v, want all optional capabilities", snapshot.Capabilities)
	}
	if snapshot.Status.State != StateReady {
		t.Fatalf("Snapshot.Status.State = %q, want %q", snapshot.Status.State, StateReady)
	}
	if snapshot.Readiness == nil {
		t.Fatal("Snapshot.Readiness is nil, want readiness")
	}
	if snapshot.Readiness.Reason != "ready" {
		t.Fatalf("Snapshot.Readiness.Reason = %q, want ready", snapshot.Readiness.Reason)
	}
	if snapshot.Inspection == nil {
		t.Fatal("Snapshot.Inspection is nil, want inspection")
	}
	if snapshot.Inspection.Summary != "ok" {
		t.Fatalf("Snapshot.Inspection.Summary = %v, want ok", snapshot.Inspection.Summary)
	}
}

func TestRegistrySnapshotIncludesInspectionError(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := errorInspectorComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
		err:  errors.New("inspection failed"),
	}

	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	snapshot, err := registry.Snapshot(ctx, "component")
	if err != nil {
		t.Fatalf("Snapshot error = %v", err)
	}
	if snapshot.Component.Name != "component" {
		t.Fatalf("Snapshot.Component.Name = %q, want component", snapshot.Component.Name)
	}
	if !snapshot.Capabilities.Inspector {
		t.Fatal("Snapshot.Capabilities.Inspector = false, want true")
	}
	if snapshot.Status.State != StateReady {
		t.Fatalf("Snapshot.Status.State = %q, want %q", snapshot.Status.State, StateReady)
	}
	if snapshot.Readiness == nil {
		t.Fatal("Snapshot.Readiness is nil, want readiness")
	}
	if snapshot.Inspection != nil {
		t.Fatalf("Snapshot.Inspection = %+v, want nil", snapshot.Inspection)
	}
	if snapshot.InspectionError != "inspection failed" {
		t.Fatalf("Snapshot.InspectionError = %q, want inspection failed", snapshot.InspectionError)
	}
}

func TestRegistrySnapshotIncludesInformationalInspection(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := &testOperationalComponent{
		info:       ComponentInfo{Name: "component", Kind: "test"},
		status:     ReadyStatus("ready"),
		readiness:  NotReadyReadiness("informational readiness"),
		inspection: Inspection{Summary: "ok"},
	}

	if err := registry.Register(component, Informational()); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	snapshot, err := registry.Snapshot(ctx, "component")
	if err != nil {
		t.Fatalf("Snapshot error = %v", err)
	}
	if snapshot.Readiness != nil {
		t.Fatal("Snapshot.Readiness is not nil, want nil")
	}
	if snapshot.Inspection == nil {
		t.Fatal("Snapshot.Inspection is nil, want inspection")
	}
	if snapshot.Inspection.Summary != "ok" {
		t.Fatalf("Snapshot.Inspection.Summary = %v, want ok", snapshot.Inspection.Summary)
	}
}

func TestRegistrySnapshotIncludesInformationalInspectionError(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := errorInspectorComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
		err:  errors.New("inspection failed"),
	}

	if err := registry.Register(component, Informational()); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	snapshot, err := registry.Snapshot(ctx, "component")
	if err != nil {
		t.Fatalf("Snapshot error = %v", err)
	}
	if snapshot.Readiness != nil {
		t.Fatal("Snapshot.Readiness is not nil, want nil")
	}
	if !snapshot.Capabilities.Inspector {
		t.Fatal("Snapshot.Capabilities.Inspector = false, want true")
	}
	if snapshot.Inspection != nil {
		t.Fatalf("Snapshot.Inspection = %+v, want nil", snapshot.Inspection)
	}
	if snapshot.InspectionError != "inspection failed" {
		t.Fatalf("Snapshot.InspectionError = %q, want inspection failed", snapshot.InspectionError)
	}
}

func TestRegistrySnapshotErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	registry := NewRegistry()
	component := testComponent{
		info:   ComponentInfo{Name: "component", Kind: "test"},
		status: ReadyStatus("ready"),
	}
	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	if _, err := registry.Snapshot(context.Background(), "missing"); err != ErrComponentNotFound {
		t.Fatalf("Snapshot(missing) error = %v, want %v", err, ErrComponentNotFound)
	}
	if _, err := registry.Snapshot(ctx, "component"); err != context.Canceled {
		t.Fatalf("Snapshot(canceled) error = %v, want %v", err, context.Canceled)
	}
}

func TestRegistryInspect(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	inspector := &testOperationalComponent{
		info:       ComponentInfo{Name: "inspector", Kind: "test"},
		status:     ReadyStatus("ready"),
		inspection: Inspection{Summary: "ok"},
	}
	plain := testComponent{
		info:   ComponentInfo{Name: "plain", Kind: "test"},
		status: ReadyStatus("ready"),
	}

	if err := registry.Register(inspector); err != nil {
		t.Fatalf("Register(inspector) error = %v", err)
	}
	if err := registry.Register(plain); err != nil {
		t.Fatalf("Register(plain) error = %v", err)
	}

	inspection, err := registry.Inspect(ctx, "inspector")
	if err != nil {
		t.Fatalf("Inspect(inspector) error = %v", err)
	}
	if inspection.Summary != "ok" {
		t.Fatalf("Inspection.Summary = %v, want ok", inspection.Summary)
	}

	if _, err := registry.Inspect(ctx, "plain"); err != ErrInspectionUnsupported {
		t.Fatalf("Inspect(plain) error = %v, want %v", err, ErrInspectionUnsupported)
	}
	if _, err := registry.Inspect(ctx, "missing"); err != ErrComponentNotFound {
		t.Fatalf("Inspect(missing) error = %v, want %v", err, ErrComponentNotFound)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := registry.Inspect(canceled, "inspector"); err != context.Canceled {
		t.Fatalf("Inspect(canceled) error = %v, want %v", err, context.Canceled)
	}
}

func TestRegistryInspectReturnsInspectorError(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	want := errors.New("inspection failed")
	component := errorInspectorComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
		err:  want,
	}

	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	if _, err := registry.Inspect(ctx, "component"); err != want {
		t.Fatalf("Inspect error = %v, want %v", err, want)
	}
}

func TestRegistryCapabilityAccessors(t *testing.T) {
	registry := NewRegistry()

	operational := &testOperationalComponent{
		info:       ComponentInfo{Name: "operational", Kind: "test"},
		status:     ReadyStatus("ready"),
		readiness:  ReadyReadiness("ready"),
		inspection: Inspection{Summary: "ok"},
		checks: []CheckDescriptor{
			{
				Name:        "cache",
				Kind:        "dependency",
				Description: "ping cache",
				Attributes: []Attribute{
					Attr("target", "cache"),
				},
			},
		},
		commands: []CommandDescriptor{
			{
				Name:        "test/run",
				Description: "run test command",
				Idempotent:  true,
				Attributes: []Attribute{
					Attr("scope", "test"),
				},
			},
		},
	}
	plain := testComponent{
		info:   ComponentInfo{Name: "plain", Kind: "test"},
		status: ReadyStatus("ready"),
	}

	if err := registry.Register(operational); err != nil {
		t.Fatalf("Register(operational) error = %v", err)
	}
	if err := registry.Register(plain); err != nil {
		t.Fatalf("Register(plain) error = %v", err)
	}

	if _, err := registry.Checker("operational"); err != nil {
		t.Fatalf("Checker(operational) error = %v", err)
	}
	if _, err := registry.CheckDescriber("operational"); err != nil {
		t.Fatalf("CheckDescriber(operational) error = %v", err)
	}
	checks, err := registry.Checks(context.Background(), "operational")
	if err != nil {
		t.Fatalf("Checks(operational) error = %v", err)
	}
	if len(checks) != 1 || checks[0].Name != "cache" {
		t.Fatalf("Checks(operational) = %+v, want cache check", checks)
	}
	checks[0].Name = "mutated"
	checks[0].Attributes[0] = Attr("mutated", "true")
	checks, err = registry.Checks(context.Background(), "operational")
	if err != nil {
		t.Fatalf("Checks(operational) second call error = %v", err)
	}
	if checks[0].Name != "cache" {
		t.Fatalf("Checks returned mutable check descriptors, name = %q", checks[0].Name)
	}
	if checks[0].Attributes[0] != Attr("target", "cache") {
		t.Fatalf("Checks returned mutable check attributes, attributes = %+v", checks[0].Attributes)
	}
	if _, err := registry.CheckGroup("operational"); err != nil {
		t.Fatalf("CheckGroup(operational) error = %v", err)
	}
	if _, err := registry.CommandHandler("operational"); err != nil {
		t.Fatalf("CommandHandler(operational) error = %v", err)
	}
	if _, err := registry.CommandDescriber("operational"); err != nil {
		t.Fatalf("CommandDescriber(operational) error = %v", err)
	}
	commands, err := registry.Commands(context.Background(), "operational")
	if err != nil {
		t.Fatalf("Commands(operational) error = %v", err)
	}
	if len(commands) != 1 || commands[0].Name != "test/run" {
		t.Fatalf("Commands(operational) = %+v, want test/run command", commands)
	}
	commands[0].Name = "mutated"
	commands[0].Attributes[0] = Attr("mutated", "true")
	commands, err = registry.Commands(context.Background(), "operational")
	if err != nil {
		t.Fatalf("Commands(operational) second call error = %v", err)
	}
	if commands[0].Name != "test/run" {
		t.Fatalf("Commands returned mutable command descriptors, name = %q", commands[0].Name)
	}
	if commands[0].Attributes[0] != Attr("scope", "test") {
		t.Fatalf("Commands returned mutable command attributes, attributes = %+v", commands[0].Attributes)
	}

	if _, err := registry.Checker("plain"); err != ErrCheckerUnsupported {
		t.Fatalf("Checker(plain) error = %v, want %v", err, ErrCheckerUnsupported)
	}
	if _, err := registry.CheckDescriber("plain"); err != ErrCheckDescriberUnsupported {
		t.Fatalf("CheckDescriber(plain) error = %v, want %v", err, ErrCheckDescriberUnsupported)
	}
	if _, err := registry.Checks(context.Background(), "plain"); err != ErrCheckDescriberUnsupported {
		t.Fatalf("Checks(plain) error = %v, want %v", err, ErrCheckDescriberUnsupported)
	}
	if _, err := registry.CheckGroup("plain"); err != ErrCheckGroupUnsupported {
		t.Fatalf("CheckGroup(plain) error = %v, want %v", err, ErrCheckGroupUnsupported)
	}
	if _, err := registry.CommandHandler("plain"); err != ErrCommandHandlerUnsupported {
		t.Fatalf("CommandHandler(plain) error = %v, want %v", err, ErrCommandHandlerUnsupported)
	}
	if _, err := registry.CommandDescriber("plain"); err != ErrCommandDescriberUnsupported {
		t.Fatalf("CommandDescriber(plain) error = %v, want %v", err, ErrCommandDescriberUnsupported)
	}
	if _, err := registry.Commands(context.Background(), "plain"); err != ErrCommandDescriberUnsupported {
		t.Fatalf("Commands(plain) error = %v, want %v", err, ErrCommandDescriberUnsupported)
	}

	if _, err := registry.Checker("missing"); err != ErrComponentNotFound {
		t.Fatalf("Checker(missing) error = %v, want %v", err, ErrComponentNotFound)
	}
	if _, err := registry.CheckDescriber("missing"); err != ErrComponentNotFound {
		t.Fatalf("CheckDescriber(missing) error = %v, want %v", err, ErrComponentNotFound)
	}
	if _, err := registry.Checks(context.Background(), "missing"); err != ErrComponentNotFound {
		t.Fatalf("Checks(missing) error = %v, want %v", err, ErrComponentNotFound)
	}
	if _, err := registry.CheckGroup("missing"); err != ErrComponentNotFound {
		t.Fatalf("CheckGroup(missing) error = %v, want %v", err, ErrComponentNotFound)
	}
	if _, err := registry.CommandHandler("missing"); err != ErrComponentNotFound {
		t.Fatalf("CommandHandler(missing) error = %v, want %v", err, ErrComponentNotFound)
	}
	if _, err := registry.CommandDescriber("missing"); err != ErrComponentNotFound {
		t.Fatalf("CommandDescriber(missing) error = %v, want %v", err, ErrComponentNotFound)
	}
	if _, err := registry.Commands(context.Background(), "missing"); err != ErrComponentNotFound {
		t.Fatalf("Commands(missing) error = %v, want %v", err, ErrComponentNotFound)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := registry.Checks(canceled, "operational"); err != context.Canceled {
		t.Fatalf("Checks(canceled) error = %v, want %v", err, context.Canceled)
	}
	if _, err := registry.Commands(canceled, "operational"); err != context.Canceled {
		t.Fatalf("Commands(canceled) error = %v, want %v", err, context.Canceled)
	}
}

func TestRegistryReadinessWithOnlyOptionalComponentsIsNotReady(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := testComponent{
		info:   ComponentInfo{Name: "optional", Kind: "test"},
		status: ReadyStatus("optional ready"),
	}

	if err := registry.Register(component, Optional()); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	readiness := registry.Readiness(ctx)
	if readiness.Ready {
		t.Fatal("Readiness.Ready = true, want false")
	}
	if readiness.Reason != "no required readiness components registered" {
		t.Fatalf("Readiness.Reason = %q, want no required readiness components registered", readiness.Reason)
	}
	if len(readiness.Components) != 1 {
		t.Fatalf("Readiness.Components length = %d, want 1", len(readiness.Components))
	}
}

func TestWithReadinessPolicyDefaultsUnknownPolicyToRequired(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := testComponent{
		info:   ComponentInfo{Name: "component", Kind: "test"},
		status: ReadyStatus("ready"),
	}

	if err := registry.Register(component, WithReadinessPolicy(ReadinessPolicy("invalid"))); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	status := registry.Status(ctx)
	if got := status.Components[0].Registration.ReadinessPolicy; got != ReadinessRequired {
		t.Fatalf("ReadinessPolicy = %q, want %q", got, ReadinessRequired)
	}
}

func TestRegistryRegisterValidatesComponentNames(t *testing.T) {
	tests := []struct {
		name string
		want error
	}{
		{name: "", want: ErrEmptyComponentName},
		{name: "   ", want: ErrEmptyComponentName},
		{name: " worker", want: ErrInvalidComponentName},
		{name: "worker ", want: ErrInvalidComponentName},
		{name: "worker one", want: ErrInvalidComponentName},
		{name: "runtime/worker", want: ErrInvalidComponentName},
		{name: "../config", want: ErrInvalidComponentName},
		{name: ".", want: ErrInvalidComponentName},
		{name: "..", want: ErrInvalidComponentName},
		{name: "worker:one", want: ErrInvalidComponentName},
		{name: "worker@one", want: ErrInvalidComponentName},
		{name: "worker_1.2-alpha", want: nil},
		{name: "WorkerA", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			component := testComponent{
				info:   ComponentInfo{Name: tt.name, Kind: "test"},
				status: ReadyStatus("ready"),
			}

			err := registry.Register(component)
			if err != tt.want {
				t.Fatalf("Register error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestRegistryReadModelsUseRegisteredComponentInfo(t *testing.T) {
	ctx := context.Background()
	component := &countingComponent{
		info: ComponentInfo{
			Name:        "component",
			Kind:        "test",
			Description: "registered description",
		},
		status: ReadyStatus("ready"),
	}

	registry := NewRegistry()
	if err := registry.Register(component, Optional()); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	component.info = ComponentInfo{
		Name:        "mutated component",
		Kind:        "mutated",
		Description: "mutated description",
	}

	status := registry.Status(ctx)
	if len(status.Components) != 1 {
		t.Fatalf("Status.Components length = %d, want 1", len(status.Components))
	}
	if got := status.Components[0].Component; got != (ComponentInfo{Name: "component", Kind: "test", Description: "registered description"}) {
		t.Fatalf("Status.Component = %+v, want registered component info", got)
	}

	readiness := registry.Readiness(ctx)
	if len(readiness.Components) != 1 {
		t.Fatalf("Readiness.Components length = %d, want 1", len(readiness.Components))
	}
	if got := readiness.Components[0].Name; got != "component" {
		t.Fatalf("Readiness.Components[0].Name = %q, want component", got)
	}
	if got := readiness.Components[0].Kind; got != "test" {
		t.Fatalf("Readiness.Components[0].Kind = %q, want test", got)
	}

	snapshot, err := registry.Snapshot(ctx, "component")
	if err != nil {
		t.Fatalf("Snapshot error = %v", err)
	}
	if got := snapshot.Component; got != (ComponentInfo{Name: "component", Kind: "test", Description: "registered description"}) {
		t.Fatalf("Snapshot.Component = %+v, want registered component info", got)
	}
	if snapshot.Readiness == nil {
		t.Fatal("Snapshot.Readiness is nil, want readiness")
	}
	if got := snapshot.Readiness.Components[0].Name; got != "component" {
		t.Fatalf("Snapshot.Readiness.Components[0].Name = %q, want component", got)
	}
	if got := snapshot.Readiness.Components[0].Kind; got != "test" {
		t.Fatalf("Snapshot.Readiness.Components[0].Kind = %q, want test", got)
	}
}

type testComponent struct {
	info   ComponentInfo
	status Status
}

func (c testComponent) ComponentInfo() ComponentInfo {
	return c.info
}

func (c testComponent) Status(context.Context) Status {
	return c.status
}

type errorInspectorComponent struct {
	info ComponentInfo
	err  error
}

func (c errorInspectorComponent) ComponentInfo() ComponentInfo {
	return c.info
}

func (c errorInspectorComponent) Status(context.Context) Status {
	return ReadyStatus("ready")
}

func (c errorInspectorComponent) Inspect(context.Context) (Inspection, error) {
	return Inspection{}, c.err
}

type testReadinessComponent struct {
	info           ComponentInfo
	status         Status
	readiness      Readiness
	readinessCalls int
}

func (c *testReadinessComponent) ComponentInfo() ComponentInfo {
	return c.info
}

func (c *testReadinessComponent) Status(context.Context) Status {
	return c.status
}

func (c *testReadinessComponent) Readiness(context.Context) Readiness {
	c.readinessCalls++
	return c.readiness
}

type countingComponent struct {
	info        ComponentInfo
	status      Status
	statusCalls int
}

func (c *countingComponent) ComponentInfo() ComponentInfo {
	return c.info
}

func (c *countingComponent) Status(context.Context) Status {
	c.statusCalls++
	return c.status
}

type testOperationalComponent struct {
	info       ComponentInfo
	status     Status
	readiness  Readiness
	inspection Inspection
	checks     []CheckDescriptor
	commands   []CommandDescriptor
}

func (c *testOperationalComponent) ComponentInfo() ComponentInfo {
	return c.info
}

func (c *testOperationalComponent) Status(context.Context) Status {
	return c.status
}

func (c *testOperationalComponent) Readiness(context.Context) Readiness {
	return c.readiness
}

func (c *testOperationalComponent) Inspect(context.Context) (Inspection, error) {
	return c.inspection, nil
}

func (c *testOperationalComponent) Check(context.Context) CheckResult {
	return ReadyCheck("ready", 0)
}

func (c *testOperationalComponent) Checks(context.Context) []CheckDescriptor {
	if c.checks != nil {
		return c.checks
	}

	return []CheckDescriptor{
		{
			Name:        "cache",
			Kind:        "dependency",
			Description: "ping cache",
			Attributes: []Attribute{
				Attr("target", "cache"),
			},
		},
	}
}

func (c *testOperationalComponent) CheckAll(context.Context) CheckSummary {
	return CheckSummary{
		State: StateReady,
		Ready: true,
	}
}

func (c *testOperationalComponent) HandleCommand(context.Context, CommandRequest) CommandResult {
	return CompletedCommand("completed", nil, 0)
}

func (c *testOperationalComponent) Commands(context.Context) []CommandDescriptor {
	if c.commands != nil {
		return c.commands
	}

	return []CommandDescriptor{
		{
			Name:        "test/run",
			Description: "run test command",
			Idempotent:  true,
			Attributes: []Attribute{
				Attr("scope", "test"),
			},
		},
	}
}
