package opskit

import (
	"context"
	"errors"
	"testing"
)

func TestRegistryHelperRegistrationAndSnapshot(t *testing.T) {
	registry := NewRegistry()

	first := testComponent{
		info:   ComponentInfo{Name: "first", Kind: "test"},
		status: ReadyStatus("first ready"),
	}
	second := testComponent{
		info:   ComponentInfo{Name: "second", Kind: "test"},
		status: ReadyStatus("second ready"),
	}

	if _, ok := registry.registration("missing"); ok {
		t.Fatal("registration(missing) ok = true, want false")
	}

	if err := registry.Register(first); err != nil {
		t.Fatalf("Register(first) error = %v", err)
	}
	if err := registry.Register(second, Optional()); err != nil {
		t.Fatalf("Register(second) error = %v", err)
	}

	reg, ok := registry.registration("second")
	if !ok {
		t.Fatal("registration(second) ok = false, want true")
	}
	if reg.info.Name != "second" {
		t.Fatalf("registration(second).info.Name = %q, want second", reg.info.Name)
	}
	if reg.readinessPolicy != ReadinessOptional {
		t.Fatalf("registration(second).readinessPolicy = %q, want %q", reg.readinessPolicy, ReadinessOptional)
	}

	registrations := registry.snapshot()
	if len(registrations) != 2 {
		t.Fatalf("snapshot length = %d, want 2", len(registrations))
	}
	if got := registrations[0].info.Name; got != "first" {
		t.Fatalf("snapshot[0].info.Name = %q, want first", got)
	}
	if got := registrations[1].info.Name; got != "second" {
		t.Fatalf("snapshot[1].info.Name = %q, want second", got)
	}

	registrations[0] = registrations[1]
	registrations = registry.snapshot()
	if got := registrations[0].info.Name; got != "first" {
		t.Fatalf("snapshot returned mutable registry slice, first info name = %q", got)
	}
}

func TestRegistryHelperEnsureInitializedLocked(t *testing.T) {
	var registry Registry

	registry.mu.Lock()
	registry.ensureInitializedLocked()
	registry.mu.Unlock()

	if registry.registrations == nil {
		t.Fatal("registrations is nil, want initialized map")
	}
}

func TestIsValidComponentName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "", want: false},
		{name: "component", want: true},
		{name: "Component_1.2-alpha", want: true},
		{name: " component", want: false},
		{name: "component ", want: false},
		{name: ".", want: false},
		{name: "..", want: false},
		{name: "component/name", want: false},
		{name: "component name", want: false},
		{name: "component:name", want: false},
		{name: "component@name", want: false},
		{name: "café", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidComponentName(tt.name); got != tt.want {
				t.Fatalf("isValidComponentName(%q) = %t, want %t", tt.name, got, tt.want)
			}
		})
	}
}

func TestCapabilitiesOf(t *testing.T) {
	plain := capabilitiesOf(testComponent{
		info:   ComponentInfo{Name: "plain", Kind: "test"},
		status: ReadyStatus("ready"),
	})
	if plain != (ComponentCapabilities{}) {
		t.Fatalf("capabilitiesOf(plain) = %+v, want zero capabilities", plain)
	}

	full := capabilitiesOf(&testOperationalComponent{
		info:       ComponentInfo{Name: "full", Kind: "test"},
		status:     ReadyStatus("ready"),
		readiness:  ReadyReadiness("ready"),
		inspection: Inspection{Summary: "ok"},
	})
	if !full.ReadinessContributor ||
		!full.Inspector ||
		!full.Checker ||
		!full.CheckGroup ||
		!full.CommandHandler ||
		!full.CommandDescriber {
		t.Fatalf("capabilitiesOf(full) = %+v, want all optional capabilities", full)
	}
}

func TestNormalizeReadinessPolicy(t *testing.T) {
	tests := []struct {
		name   string
		policy ReadinessPolicy
		want   ReadinessPolicy
	}{
		{name: "required", policy: ReadinessRequired, want: ReadinessRequired},
		{name: "optional", policy: ReadinessOptional, want: ReadinessOptional},
		{name: "informational", policy: ReadinessInformational, want: ReadinessInformational},
		{name: "unknown", policy: ReadinessPolicy("unknown"), want: ReadinessRequired},
		{name: "empty", policy: "", want: ReadinessRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeReadinessPolicy(tt.policy); got != tt.want {
				t.Fatalf("normalizeReadinessPolicy(%q) = %q, want %q", tt.policy, got, tt.want)
			}
		})
	}
}

func TestParticipatesInReadiness(t *testing.T) {
	tests := []struct {
		name   string
		policy ReadinessPolicy
		want   bool
	}{
		{name: "required", policy: ReadinessRequired, want: true},
		{name: "optional", policy: ReadinessOptional, want: true},
		{name: "informational", policy: ReadinessInformational, want: false},
		{name: "unknown defaults required", policy: ReadinessPolicy("unknown"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := participatesInReadiness(tt.policy); got != tt.want {
				t.Fatalf("participatesInReadiness(%q) = %t, want %t", tt.policy, got, tt.want)
			}
		})
	}
}

func TestBlocksReadiness(t *testing.T) {
	tests := []struct {
		name   string
		policy ReadinessPolicy
		want   bool
	}{
		{name: "required", policy: ReadinessRequired, want: true},
		{name: "optional", policy: ReadinessOptional, want: false},
		{name: "informational", policy: ReadinessInformational, want: false},
		{name: "unknown defaults required", policy: ReadinessPolicy("unknown"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := blocksReadiness(tt.policy); got != tt.want {
				t.Fatalf("blocksReadiness(%q) = %t, want %t", tt.policy, got, tt.want)
			}
		})
	}
}

func TestReadinessItemFromReadinessFallback(t *testing.T) {
	items := readinessItemFromReadiness(
		ComponentInfo{Name: "component", Kind: "test"},
		NotReadyReadiness("not ready"),
		ReadinessRequired,
	)

	if len(items) != 1 {
		t.Fatalf("items length = %d, want 1", len(items))
	}
	item := items[0]
	if item.Name != "component" {
		t.Fatalf("Name = %q, want component", item.Name)
	}
	if item.Kind != "test" {
		t.Fatalf("Kind = %q, want test", item.Kind)
	}
	if item.Policy != ReadinessRequired {
		t.Fatalf("Policy = %q, want %q", item.Policy, ReadinessRequired)
	}
	if item.Ready {
		t.Fatal("Ready = true, want false")
	}
	if item.State != StateNotReady {
		t.Fatalf("State = %q, want %q", item.State, StateNotReady)
	}
	if item.Reason != "not ready" {
		t.Fatalf("Reason = %q, want not ready", item.Reason)
	}
}

func TestReadinessItemFromReadinessNormalizesContributorItems(t *testing.T) {
	input := Readiness{
		Ready:  true,
		Reason: "ready",
		Components: []ReadinessItem{
			{Ready: true},
			{Name: "child", Kind: "dependency", Ready: false, State: StateFailed, Policy: ReadinessRequired},
		},
	}

	items := readinessItemFromReadiness(
		ComponentInfo{Name: "component", Kind: "test"},
		input,
		ReadinessOptional,
	)

	if len(items) != 2 {
		t.Fatalf("items length = %d, want 2", len(items))
	}
	if items[0].Name != "component" {
		t.Fatalf("items[0].Name = %q, want component", items[0].Name)
	}
	if items[0].Kind != "test" {
		t.Fatalf("items[0].Kind = %q, want test", items[0].Kind)
	}
	if items[0].Policy != ReadinessOptional {
		t.Fatalf("items[0].Policy = %q, want %q", items[0].Policy, ReadinessOptional)
	}
	if items[0].State != StateReady {
		t.Fatalf("items[0].State = %q, want %q", items[0].State, StateReady)
	}
	if items[1].Name != "child" {
		t.Fatalf("items[1].Name = %q, want child", items[1].Name)
	}
	if items[1].Kind != "dependency" {
		t.Fatalf("items[1].Kind = %q, want dependency", items[1].Kind)
	}
	if items[1].Policy != ReadinessRequired {
		t.Fatalf("items[1].Policy = %q, want %q", items[1].Policy, ReadinessRequired)
	}
	if items[1].State != StateFailed {
		t.Fatalf("items[1].State = %q, want %q", items[1].State, StateFailed)
	}
	if input.Components[1].Policy != ReadinessRequired {
		t.Fatalf("input component policy mutated to %q, want %q", input.Components[1].Policy, ReadinessRequired)
	}
}

func TestCanceledComponentStatus(t *testing.T) {
	err := context.Canceled
	status := canceledComponentStatus(err)

	if status.Component.Name != "opskit.registry" {
		t.Fatalf("Component.Name = %q, want opskit.registry", status.Component.Name)
	}
	if status.Component.Kind != "opskit" {
		t.Fatalf("Component.Kind = %q, want opskit", status.Component.Kind)
	}
	if status.Status.State != StateUnknown {
		t.Fatalf("Status.State = %q, want %q", status.Status.State, StateUnknown)
	}
	if status.Status.Ready {
		t.Fatal("Status.Ready = true, want false")
	}
	if status.Status.Message != "status evaluation canceled" {
		t.Fatalf("Status.Message = %q, want status evaluation canceled", status.Status.Message)
	}
	if len(status.Status.Attributes) != 1 || status.Status.Attributes[0] != Attr("error", err.Error()) {
		t.Fatalf("Status.Attributes = %+v, want error attribute", status.Status.Attributes)
	}
}

func TestCanceledReadinessItem(t *testing.T) {
	err := errors.New("deadline exceeded")
	item := canceledReadinessItem(err)

	if item.Name != "opskit.registry" {
		t.Fatalf("Name = %q, want opskit.registry", item.Name)
	}
	if item.Kind != "opskit" {
		t.Fatalf("Kind = %q, want opskit", item.Kind)
	}
	if item.Ready {
		t.Fatal("Ready = true, want false")
	}
	if item.State != StateUnknown {
		t.Fatalf("State = %q, want %q", item.State, StateUnknown)
	}
	if item.Reason != "readiness evaluation canceled" {
		t.Fatalf("Reason = %q, want readiness evaluation canceled", item.Reason)
	}
	if item.Message != err.Error() {
		t.Fatalf("Message = %q, want %q", item.Message, err.Error())
	}
}
