package opskit

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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

func TestRegistryEntries(t *testing.T) {
	var registry Registry

	first := testComponent{
		info: ComponentInfo{
			Name: "first",
			Kind: "test",
			Labels: []Attribute{
				Attr("tier", "critical"),
			},
		},
		status: ReadyStatus("first ready"),
	}
	second := &testOperationalComponent{
		info: ComponentInfo{
			Name: "second",
			Kind: "operational",
			Labels: []Attribute{
				Attr("scope", "admin"),
			},
		},
		status: ReadyStatus("second ready"),
	}

	if err := registry.Register(first); err != nil {
		t.Fatalf("Register(first) error = %v", err)
	}
	if err := registry.Register(second, Optional()); err != nil {
		t.Fatalf("Register(second) error = %v", err)
	}

	entries := registry.Entries()

	if len(entries) != 2 {
		t.Fatalf("Entries length = %d, want 2", len(entries))
	}
	requireComponentInfo(t, entries[0].Component, first.info)
	if entries[0].Registration.ReadinessPolicy != ReadinessRequired {
		t.Fatalf("Entries[0].Registration.ReadinessPolicy = %q, want %q", entries[0].Registration.ReadinessPolicy, ReadinessRequired)
	}
	if entries[0].Capabilities != (ComponentCapabilities{}) {
		t.Fatalf("Entries[0].Capabilities = %+v, want none", entries[0].Capabilities)
	}
	requireComponentInfo(t, entries[1].Component, second.info)
	if entries[1].Registration.ReadinessPolicy != ReadinessOptional {
		t.Fatalf("Entries[1].Registration.ReadinessPolicy = %q, want %q", entries[1].Registration.ReadinessPolicy, ReadinessOptional)
	}
	if !entries[1].Capabilities.ReadinessContributor ||
		!entries[1].Capabilities.Inspector ||
		!entries[1].Capabilities.Checker ||
		!entries[1].Capabilities.CheckDescriber ||
		!entries[1].Capabilities.CheckGroup ||
		!entries[1].Capabilities.CommandHandler ||
		!entries[1].Capabilities.CommandDescriber {
		t.Fatalf("Entries[1].Capabilities = %+v, want all optional capabilities", entries[1].Capabilities)
	}

	entries[0].Component.Name = "mutated"
	entries[0].Component.Labels[0] = Attr("tier", "mutated")
	entries = registry.Entries()
	if entries[0].Component.Name != "first" {
		t.Fatalf("Entries returned mutable registry slice, first name = %q", entries[0].Component.Name)
	}
	if entries[0].Component.Labels[0] != Attr("tier", "critical") {
		t.Fatalf("Entries returned mutable component labels, labels = %+v", entries[0].Component.Labels)
	}
}

func TestRegistryEntriesEmpty(t *testing.T) {
	var registry Registry

	entries := registry.Entries()
	if len(entries) != 0 {
		t.Fatalf("Entries length = %d, want 0", len(entries))
	}
	if entries == nil {
		t.Fatal("Entries = nil, want empty slice")
	}
}

func TestRegistryEntriesDoesNotInvokeComponentMethods(t *testing.T) {
	var registry Registry

	component := &countingComponent{
		info:   ComponentInfo{Name: "component", Kind: "test"},
		status: ReadyStatus("ready"),
	}

	if err := registry.Register(component); err != nil {
		t.Fatalf("Register(component) error = %v", err)
	}

	entries := registry.Entries()

	if len(entries) != 1 {
		t.Fatalf("Entries length = %d, want 1", len(entries))
	}
	if component.statusCalls != 0 {
		t.Fatalf("Status calls = %d, want 0", component.statusCalls)
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

func TestRegistryStatusRecoversComponentPanic(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	panicking := &panicComponent{
		info:        ComponentInfo{Name: "panicking", Kind: "test"},
		statusPanic: "secret status panic",
	}
	ready := testComponent{
		info:   ComponentInfo{Name: "ready", Kind: "test"},
		status: ReadyStatus("ready"),
	}

	if err := registry.Register(panicking); err != nil {
		t.Fatalf("Register(panicking) error = %v", err)
	}
	if err := registry.Register(ready); err != nil {
		t.Fatalf("Register(ready) error = %v", err)
	}

	status := registry.Status(ctx)
	if len(status.Components) != 2 {
		t.Fatalf("Status.Components length = %d, want 2", len(status.Components))
	}
	if got := status.Components[0].Status.State; got != StateUnknown {
		t.Fatalf("panicking status state = %q, want %q", got, StateUnknown)
	}
	if status.Components[0].Status.Ready {
		t.Fatal("panicking status ready = true, want false")
	}
	if got := status.Components[0].Status.Message; got != componentStatusPanicMessage {
		t.Fatalf("panicking status message = %q, want %q", got, componentStatusPanicMessage)
	}
	if strings.Contains(status.Components[0].Status.Message, "secret") {
		t.Fatalf("panic value exposed in status message %q", status.Components[0].Status.Message)
	}
	if got := status.Components[1].Status.State; got != StateReady {
		t.Fatalf("ready status state = %q, want %q", got, StateReady)
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
	if got := readiness.Components[0].Component.Name; got != "opskit.registry" {
		t.Fatalf("Readiness.Components[0].Component.Name = %q, want opskit.registry", got)
	}
}

func TestRegistryReadinessRecoversComponentPanic(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	required := &panicComponent{
		info:           ComponentInfo{Name: "required", Kind: "test"},
		status:         ReadyStatus("ready"),
		readinessPanic: "secret readiness panic",
	}
	optional := &panicStatusOnlyComponent{
		info:  ComponentInfo{Name: "optional", Kind: "test"},
		panic: "secret status panic",
	}

	if err := registry.Register(required); err != nil {
		t.Fatalf("Register(required) error = %v", err)
	}
	if err := registry.Register(optional, Optional()); err != nil {
		t.Fatalf("Register(optional) error = %v", err)
	}

	readiness := registry.Readiness(ctx)
	if readiness.Ready {
		t.Fatal("Readiness.Ready = true, want false")
	}
	if readiness.Reason != "one or more required readiness components are not ready" {
		t.Fatalf("Readiness.Reason = %q, want one or more required readiness components are not ready", readiness.Reason)
	}
	if len(readiness.Components) != 2 {
		t.Fatalf("Readiness.Components length = %d, want 2", len(readiness.Components))
	}
	if got := readiness.Components[0].Readiness.Items[0].State; got != StateUnknown {
		t.Fatalf("required readiness state = %q, want %q", got, StateUnknown)
	}
	if got := readiness.Components[0].Readiness.Reason; got != componentReadinessPanicMessage {
		t.Fatalf("required readiness reason = %q, want %q", got, componentReadinessPanicMessage)
	}
	if got := readiness.Components[1].Readiness.Items[0].State; got != StateUnknown {
		t.Fatalf("optional readiness state = %q, want %q", got, StateUnknown)
	}
	if got := readiness.Components[1].Readiness.Reason; got != componentStatusPanicMessage {
		t.Fatalf("optional readiness reason = %q, want %q", got, componentStatusPanicMessage)
	}
	for _, component := range readiness.Components {
		if strings.Contains(component.Readiness.Reason, "secret") {
			t.Fatalf("panic value exposed in component readiness %+v", component)
		}
		for _, item := range component.Readiness.Items {
			if strings.Contains(item.Reason, "secret") || strings.Contains(item.Message, "secret") {
				t.Fatalf("panic value exposed in readiness item %+v", item)
			}
		}
	}
}

func TestRegistryReadinessOptionalPanicDoesNotBlockRequiredReadiness(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	required := testComponent{
		info:   ComponentInfo{Name: "required", Kind: "test"},
		status: ReadyStatus("ready"),
	}
	optional := &panicStatusOnlyComponent{
		info:  ComponentInfo{Name: "optional", Kind: "test"},
		panic: "secret optional panic",
	}

	if err := registry.Register(required); err != nil {
		t.Fatalf("Register(required) error = %v", err)
	}
	if err := registry.Register(optional, Optional()); err != nil {
		t.Fatalf("Register(optional) error = %v", err)
	}

	readiness := registry.Readiness(ctx)
	if !readiness.Ready {
		t.Fatal("Readiness.Ready = false, want true")
	}
	if len(readiness.Components) != 2 {
		t.Fatalf("Readiness.Components length = %d, want 2", len(readiness.Components))
	}
	if readiness.Components[1].Registration.ReadinessPolicy != ReadinessOptional {
		t.Fatalf("optional policy = %q, want %q", readiness.Components[1].Registration.ReadinessPolicy, ReadinessOptional)
	}
	if readiness.Components[1].Readiness.Ready {
		t.Fatal("optional component readiness = true, want false")
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
	if readiness.Reason != "all required readiness components ready" {
		t.Fatalf("Readiness.Reason = %q, want all required readiness components ready", readiness.Reason)
	}
	if len(readiness.Components) != 2 {
		t.Fatalf("Readiness.Components length = %d, want 2", len(readiness.Components))
	}
	if readiness.Components[0].Component.Name != "required" {
		t.Fatalf("Readiness.Components[0].Component.Name = %q, want required", readiness.Components[0].Component.Name)
	}
	if readiness.Components[0].Registration.ReadinessPolicy != ReadinessRequired {
		t.Fatalf("Readiness.Components[0] policy = %q, want %q", readiness.Components[0].Registration.ReadinessPolicy, ReadinessRequired)
	}
	if readiness.Components[0].Readiness.Reason != "required ready" {
		t.Fatalf("Readiness.Components[0] reason = %q, want required ready", readiness.Components[0].Readiness.Reason)
	}
	if readiness.Components[1].Component.Name != "optional" {
		t.Fatalf("Readiness.Components[1].Component.Name = %q, want optional", readiness.Components[1].Component.Name)
	}
	if readiness.Components[1].Registration.ReadinessPolicy != ReadinessOptional {
		t.Fatalf("Readiness.Components[1] policy = %q, want %q", readiness.Components[1].Registration.ReadinessPolicy, ReadinessOptional)
	}
	if readiness.Components[1].Readiness.Ready {
		t.Fatal("Readiness.Components[1].Readiness.Ready = true, want false")
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

func TestRegistryReadinessSeparatesParentPolicyFromChildImpact(t *testing.T) {
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
	componentReadiness := readiness.Components[0]
	if componentReadiness.Component.Name != "component" || componentReadiness.Component.Kind != "test" {
		t.Fatalf("component identity = %#v, want registered parent", componentReadiness.Component)
	}
	if componentReadiness.Registration.ReadinessPolicy != ReadinessOptional {
		t.Fatalf("parent policy = %q, want %q", componentReadiness.Registration.ReadinessPolicy, ReadinessOptional)
	}
	if len(componentReadiness.Readiness.Items) != 1 {
		t.Fatalf("child item count = %d, want 1", len(componentReadiness.Readiness.Items))
	}
	item := componentReadiness.Readiness.Items[0]
	if item.Name != "dependency" || item.Kind != "" {
		t.Fatalf("child identity = %#v, want contributor-owned name without injected parent kind", item)
	}
	if item.Impact != ReadinessImpactBlocking {
		t.Fatalf("child impact = %q, want %q", item.Impact, ReadinessImpactBlocking)
	}
}

func TestRegistryReadinessPreservesDuplicateChildNamesUnderParents(t *testing.T) {
	registry := NewRegistry()
	for _, component := range []*testReadinessComponent{
		{
			info:   ComponentInfo{Name: "clients", Kind: "client_registry"},
			status: ReadyStatus("ready"),
			readiness: ReadyReadiness("clients ready", ReadinessItem{
				Name: "payments", Ready: true, State: StateReady,
			}),
		},
		{
			info:   ComponentInfo{Name: "dependencies", Kind: "dependency_registry"},
			status: ReadyStatus("ready"),
			readiness: ReadyReadiness("dependencies ready", ReadinessItem{
				Name: "payments", Ready: true, State: StateReady,
			}),
		},
	} {
		if err := registry.Register(component); err != nil {
			t.Fatalf("Register(%q) error = %v", component.info.Name, err)
		}
	}

	readiness := registry.Readiness(context.Background())
	if len(readiness.Components) != 2 {
		t.Fatalf("component count = %d, want 2", len(readiness.Components))
	}
	for i, wantParent := range []string{"clients", "dependencies"} {
		component := readiness.Components[i]
		if component.Component.Name != wantParent {
			t.Fatalf("component %d parent = %q, want %q", i, component.Component.Name, wantParent)
		}
		if len(component.Readiness.Items) != 1 || component.Readiness.Items[0].Name != "payments" {
			t.Fatalf("component %d child items = %#v, want scoped payments item", i, component.Readiness.Items)
		}
	}
}

func TestRegistryReadinessTrustsContributorAggregateInvariant(t *testing.T) {
	registry := NewRegistry()
	component := &testReadinessComponent{
		info:   ComponentInfo{Name: "dependencies", Kind: "dependency_registry"},
		status: ReadyStatus("status ready"),
		readiness: NotReadyReadiness("aggregate quorum is not satisfied", ReadinessItem{
			Name: "payments", Ready: true, State: StateReady,
		}),
	}
	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	readiness := registry.Readiness(context.Background())
	if readiness.Ready {
		t.Fatal("system readiness = true, want false from contributor aggregate")
	}
	got := readiness.Components[0].Readiness
	if got.Ready || got.Reason != "aggregate quorum is not satisfied" {
		t.Fatalf("component readiness = %#v, want false with contributor reason", got)
	}
	if len(got.Items) != 1 || !got.Items[0].Ready {
		t.Fatalf("component items = %#v, want individually ready child", got.Items)
	}
}

func TestRegistryReadinessOptionalParentContainsBlockingChildWithoutBlockingSystem(t *testing.T) {
	registry := NewRegistry()
	required := &testReadinessComponent{
		info:      ComponentInfo{Name: "config", Kind: "config"},
		status:    ReadyStatus("ready"),
		readiness: ReadyReadiness("config ready"),
	}
	optional := &testReadinessComponent{
		info:   ComponentInfo{Name: "clients", Kind: "client_registry"},
		status: NotReadyStatus("not ready"),
		readiness: NotReadyReadiness("required client unavailable", ReadinessItem{
			Name:   "payments",
			Impact: ReadinessImpactBlocking,
			Ready:  false,
			State:  StateNotReady,
		}),
	}
	if err := registry.Register(required); err != nil {
		t.Fatalf("Register(required) error = %v", err)
	}
	if err := registry.Register(optional, Optional()); err != nil {
		t.Fatalf("Register(optional) error = %v", err)
	}

	readiness := registry.Readiness(context.Background())
	if !readiness.Ready {
		t.Fatalf("system readiness = false, want true: %#v", readiness)
	}
	got := readiness.Components[1]
	if got.Registration.ReadinessPolicy != ReadinessOptional {
		t.Fatalf("parent policy = %q, want %q", got.Registration.ReadinessPolicy, ReadinessOptional)
	}
	if got.Readiness.Ready {
		t.Fatal("optional parent readiness = true, want false")
	}
	if got.Readiness.Items[0].Impact != ReadinessImpactBlocking {
		t.Fatalf("child impact = %q, want %q", got.Readiness.Items[0].Impact, ReadinessImpactBlocking)
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
	if len(optionalSnapshot.Readiness.Items) != 1 {
		t.Fatalf("Snapshot(optional).Readiness.Items length = %d, want 1", len(optionalSnapshot.Readiness.Items))
	}
	if optionalSnapshot.Readiness.Items[0].Impact != ReadinessImpactBlocking {
		t.Fatalf("Snapshot(optional).Readiness.Items[0].Impact = %q, want %q", optionalSnapshot.Readiness.Items[0].Impact, ReadinessImpactBlocking)
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
	if len(requiredSnapshot.Readiness.Items) != 1 {
		t.Fatalf("Snapshot(required).Readiness.Items length = %d, want 1", len(requiredSnapshot.Readiness.Items))
	}
	if requiredSnapshot.Readiness.Items[0].Impact != ReadinessImpactBlocking {
		t.Fatalf("Snapshot(required).Readiness.Items[0].Impact = %q, want %q", requiredSnapshot.Readiness.Items[0].Impact, ReadinessImpactBlocking)
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

func TestRegistrySnapshotIncludesGenericInspectionFailure(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := errorInspectorComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
		err:  errors.New("inspection failed with secret=token"),
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
	if snapshot.InspectionFailure == nil || *snapshot.InspectionFailure != (Failure{Code: FailureCodeInspectionFailed, Message: componentInspectionFailureMessage}) {
		t.Fatalf("Snapshot.InspectionFailure = %+v, want generic failure", snapshot.InspectionFailure)
	}
	encoded, marshalErr := json.Marshal(snapshot)
	if marshalErr != nil {
		t.Fatalf("Marshal snapshot error = %v", marshalErr)
	}
	if strings.Contains(string(encoded), "secret=token") {
		t.Fatalf("snapshot exposed inspector error: %s", encoded)
	}
}

func TestRegistrySnapshotDoesNotFormatInspectorError(t *testing.T) {
	registry := NewRegistry()
	component := errorInspectorComponent{
		info: ComponentInfo{Name: "component", Kind: "test"},
		err:  panicOnErrorString{},
	}
	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	snapshot, err := registry.Snapshot(context.Background(), "component")
	if err != nil {
		t.Fatalf("Snapshot error = %v", err)
	}
	if snapshot.InspectionFailure == nil || snapshot.InspectionFailure.Code != FailureCodeInspectionFailed {
		t.Fatalf("Snapshot.InspectionFailure = %+v, want inspection failure", snapshot.InspectionFailure)
	}
}

func TestRegistrySnapshotRecoversStatusPanic(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := &panicStatusOnlyComponent{
		info:  ComponentInfo{Name: "component", Kind: "test"},
		panic: "secret status panic",
	}

	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	snapshot, err := registry.Snapshot(ctx, "component")
	if err != nil {
		t.Fatalf("Snapshot error = %v", err)
	}
	if snapshot.Status.State != StateUnknown {
		t.Fatalf("Snapshot.Status.State = %q, want %q", snapshot.Status.State, StateUnknown)
	}
	if snapshot.Status.Message != componentStatusPanicMessage {
		t.Fatalf("Snapshot.Status.Message = %q, want %q", snapshot.Status.Message, componentStatusPanicMessage)
	}
	if snapshot.Readiness == nil {
		t.Fatal("Snapshot.Readiness is nil, want readiness")
	}
	if snapshot.Readiness.Items[0].State != StateUnknown {
		t.Fatalf("Snapshot.Readiness.Items[0].State = %q, want %q", snapshot.Readiness.Items[0].State, StateUnknown)
	}
	if snapshot.Readiness.Items[0].Reason != componentStatusPanicMessage {
		t.Fatalf("Snapshot.Readiness.Items[0].Reason = %q, want %q", snapshot.Readiness.Items[0].Reason, componentStatusPanicMessage)
	}
	if strings.Contains(snapshot.Status.Message, "secret") ||
		strings.Contains(snapshot.Readiness.Items[0].Reason, "secret") {
		t.Fatalf("panic value exposed in snapshot %+v", snapshot)
	}
}

func TestRegistrySnapshotRecoversReadinessPanic(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := &panicComponent{
		info:           ComponentInfo{Name: "component", Kind: "test"},
		status:         ReadyStatus("ready"),
		readinessPanic: "secret readiness panic",
	}

	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	snapshot, err := registry.Snapshot(ctx, "component")
	if err != nil {
		t.Fatalf("Snapshot error = %v", err)
	}
	if snapshot.Status.State != StateReady {
		t.Fatalf("Snapshot.Status.State = %q, want %q", snapshot.Status.State, StateReady)
	}
	if snapshot.Readiness == nil {
		t.Fatal("Snapshot.Readiness is nil, want readiness")
	}
	if snapshot.Readiness.Ready {
		t.Fatal("Snapshot.Readiness.Ready = true, want false")
	}
	if snapshot.Readiness.Items[0].State != StateUnknown {
		t.Fatalf("Snapshot.Readiness.Items[0].State = %q, want %q", snapshot.Readiness.Items[0].State, StateUnknown)
	}
	if snapshot.Readiness.Items[0].Reason != componentReadinessPanicMessage {
		t.Fatalf("Snapshot.Readiness.Items[0].Reason = %q, want %q", snapshot.Readiness.Items[0].Reason, componentReadinessPanicMessage)
	}
	if strings.Contains(snapshot.Readiness.Items[0].Reason, "secret") {
		t.Fatalf("panic value exposed in snapshot readiness %+v", snapshot.Readiness.Items[0])
	}
}

func TestRegistrySnapshotRecoversInspectPanic(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := &panicComponent{
		info:         ComponentInfo{Name: "component", Kind: "test"},
		status:       ReadyStatus("ready"),
		readiness:    ReadyReadiness("ready"),
		inspectPanic: "secret inspection panic",
	}

	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	snapshot, err := registry.Snapshot(ctx, "component")
	if err != nil {
		t.Fatalf("Snapshot error = %v", err)
	}
	if snapshot.Inspection != nil {
		t.Fatalf("Snapshot.Inspection = %+v, want nil", snapshot.Inspection)
	}
	if snapshot.InspectionFailure == nil || *snapshot.InspectionFailure != (Failure{Code: FailureCodeInspectionFailed, Message: componentInspectionFailureMessage}) {
		t.Fatalf("Snapshot.InspectionFailure = %+v, want generic failure", snapshot.InspectionFailure)
	}
	encoded, marshalErr := json.Marshal(snapshot)
	if marshalErr != nil {
		t.Fatalf("Marshal snapshot error = %v", marshalErr)
	}
	if strings.Contains(string(encoded), "secret") {
		t.Fatalf("panic value exposed in inspection failure %s", encoded)
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

func TestRegistrySnapshotIncludesInformationalInspectionFailure(t *testing.T) {
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
	if snapshot.InspectionFailure == nil || *snapshot.InspectionFailure != (Failure{Code: FailureCodeInspectionFailed, Message: componentInspectionFailureMessage}) {
		t.Fatalf("Snapshot.InspectionFailure = %+v, want generic failure", snapshot.InspectionFailure)
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

func TestRegistryInspectRecoversPanic(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := &panicComponent{
		info:         ComponentInfo{Name: "component", Kind: "test"},
		status:       ReadyStatus("ready"),
		inspectPanic: "secret inspection panic",
	}

	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	_, err := registry.Inspect(ctx, "component")
	if !errors.Is(err, ErrComponentPanicked) {
		t.Fatalf("Inspect error = %v, want %v", err, ErrComponentPanicked)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("panic value exposed in inspect error %q", err.Error())
	}
}

func TestRegistryChecksRecoversPanic(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := &panicDescriptorComponent{
		info:        ComponentInfo{Name: "component", Kind: "test"},
		checksPanic: "secret checks panic",
	}

	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	checks, err := registry.Checks(ctx, "component")
	if !errors.Is(err, ErrComponentPanicked) {
		t.Fatalf("Checks error = %v, want %v", err, ErrComponentPanicked)
	}
	if checks != nil {
		t.Fatalf("Checks = %+v, want nil", checks)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("panic value exposed in checks error %q", err.Error())
	}
}

func TestRegistryCommandsRecoversPanic(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := &panicDescriptorComponent{
		info:          ComponentInfo{Name: "component", Kind: "test"},
		commandsPanic: "secret commands panic",
	}

	if err := registry.Register(component); err != nil {
		t.Fatalf("Register error = %v", err)
	}

	commands, err := registry.Commands(ctx, "component")
	if !errors.Is(err, ErrComponentPanicked) {
		t.Fatalf("Commands error = %v, want %v", err, ErrComponentPanicked)
	}
	if commands != nil {
		t.Fatalf("Commands = %+v, want nil", commands)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("panic value exposed in commands error %q", err.Error())
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
		{name: "worker\none", want: ErrInvalidComponentName},
		{name: "worker\tone", want: ErrInvalidComponentName},
		{name: "worker\x00one", want: ErrInvalidComponentName},
		{name: "\u2003worker", want: ErrInvalidComponentName},
		{name: "worker_1.2-alpha", want: nil},
		{name: "cache.primary_1-alpha", want: nil},
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
			Labels: []Attribute{
				Attr("tier", "critical"),
			},
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
		Labels: []Attribute{
			Attr("tier", "mutated"),
		},
	}

	wantInfo := ComponentInfo{
		Name:        "component",
		Kind:        "test",
		Description: "registered description",
		Labels: []Attribute{
			Attr("tier", "critical"),
		},
	}

	status := registry.Status(ctx)
	if len(status.Components) != 1 {
		t.Fatalf("Status.Components length = %d, want 1", len(status.Components))
	}
	requireComponentInfo(t, status.Components[0].Component, wantInfo)
	status.Components[0].Component.Labels[0] = Attr("tier", "mutated")

	readiness := registry.Readiness(ctx)
	if len(readiness.Components) != 1 {
		t.Fatalf("Readiness.Components length = %d, want 1", len(readiness.Components))
	}
	if got := readiness.Components[0].Component.Name; got != "component" {
		t.Fatalf("Readiness.Components[0].Component.Name = %q, want component", got)
	}
	if got := readiness.Components[0].Component.Kind; got != "test" {
		t.Fatalf("Readiness.Components[0].Component.Kind = %q, want test", got)
	}
	readiness.Components[0].Component.Labels[0] = Attr("tier", "mutated")

	snapshot, err := registry.Snapshot(ctx, "component")
	if err != nil {
		t.Fatalf("Snapshot error = %v", err)
	}
	requireComponentInfo(t, snapshot.Component, wantInfo)
	snapshot.Component.Labels[0] = Attr("tier", "mutated")
	if snapshot.Readiness == nil {
		t.Fatal("Snapshot.Readiness is nil, want readiness")
	}
	if got := snapshot.Readiness.Items[0].Name; got != "component" {
		t.Fatalf("Snapshot.Readiness.Items[0].Name = %q, want component", got)
	}
	if got := snapshot.Readiness.Items[0].Kind; got != "test" {
		t.Fatalf("Snapshot.Readiness.Items[0].Kind = %q, want test", got)
	}

	nextStatus := registry.Status(ctx)
	requireComponentInfo(t, nextStatus.Components[0].Component, wantInfo)
	nextReadiness := registry.Readiness(ctx)
	requireComponentInfo(t, nextReadiness.Components[0].Component, wantInfo)

	nextSnapshot, err := registry.Snapshot(ctx, "component")
	if err != nil {
		t.Fatalf("Snapshot after mutation error = %v", err)
	}
	requireComponentInfo(t, nextSnapshot.Component, wantInfo)
}

func requireComponentInfo(t *testing.T, got, want ComponentInfo) {
	t.Helper()

	if got.Name != want.Name {
		t.Fatalf("ComponentInfo.Name = %q, want %q", got.Name, want.Name)
	}
	if got.Kind != want.Kind {
		t.Fatalf("ComponentInfo.Kind = %q, want %q", got.Kind, want.Kind)
	}
	if got.Description != want.Description {
		t.Fatalf("ComponentInfo.Description = %q, want %q", got.Description, want.Description)
	}
	if len(got.Labels) != len(want.Labels) {
		t.Fatalf("ComponentInfo.Labels length = %d, want %d", len(got.Labels), len(want.Labels))
	}
	for i := range want.Labels {
		if got.Labels[i] != want.Labels[i] {
			t.Fatalf("ComponentInfo.Labels[%d] = %+v, want %+v", i, got.Labels[i], want.Labels[i])
		}
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

type panicOnErrorString struct{}

func (panicOnErrorString) Error() string {
	panic("inspector error was formatted")
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

type panicStatusOnlyComponent struct {
	info  ComponentInfo
	panic string
}

func (c *panicStatusOnlyComponent) ComponentInfo() ComponentInfo {
	return c.info
}

func (c *panicStatusOnlyComponent) Status(context.Context) Status {
	panic(c.panic)
}

type panicComponent struct {
	info           ComponentInfo
	status         Status
	readiness      Readiness
	inspection     Inspection
	statusPanic    string
	readinessPanic string
	inspectPanic   string
}

func (c *panicComponent) ComponentInfo() ComponentInfo {
	return c.info
}

func (c *panicComponent) Status(context.Context) Status {
	if c.statusPanic != "" {
		panic(c.statusPanic)
	}
	return c.status
}

func (c *panicComponent) Readiness(context.Context) Readiness {
	if c.readinessPanic != "" {
		panic(c.readinessPanic)
	}
	return c.readiness
}

func (c *panicComponent) Inspect(context.Context) (Inspection, error) {
	if c.inspectPanic != "" {
		panic(c.inspectPanic)
	}
	return c.inspection, nil
}

type panicDescriptorComponent struct {
	info          ComponentInfo
	checksPanic   string
	commandsPanic string
}

func (c *panicDescriptorComponent) ComponentInfo() ComponentInfo {
	return c.info
}

func (c *panicDescriptorComponent) Status(context.Context) Status {
	return ReadyStatus("ready")
}

func (c *panicDescriptorComponent) Checks(context.Context) []CheckDescriptor {
	if c.checksPanic != "" {
		panic(c.checksPanic)
	}
	return []CheckDescriptor{{Name: "check"}}
}

func (c *panicDescriptorComponent) Commands(context.Context) []CommandDescriptor {
	if c.commandsPanic != "" {
		panic(c.commandsPanic)
	}
	return []CommandDescriptor{{Name: "command"}}
}
