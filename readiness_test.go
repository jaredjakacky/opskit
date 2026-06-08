package opskit

import "testing"

func TestReadyReadiness(t *testing.T) {
	components := []ReadinessItem{
		{Name: "component", Kind: "test", Ready: true, State: StateReady},
	}

	readiness := ReadyReadiness("all ready", components...)
	components[0].Name = "mutated"

	if !readiness.Ready {
		t.Fatal("Ready = false, want true")
	}
	if readiness.Reason != "all ready" {
		t.Fatalf("Reason = %q, want all ready", readiness.Reason)
	}
	if len(readiness.Components) != 1 {
		t.Fatalf("Components length = %d, want 1", len(readiness.Components))
	}
	if readiness.Components[0].Name != "component" {
		t.Fatalf("Components[0].Name = %q, want component", readiness.Components[0].Name)
	}
}

func TestNotReadyReadiness(t *testing.T) {
	components := []ReadinessItem{
		{Name: "component", Kind: "test", Ready: false, State: StateNotReady},
	}

	readiness := NotReadyReadiness("not ready", components...)
	components[0].Name = "mutated"

	if readiness.Ready {
		t.Fatal("Ready = true, want false")
	}
	if readiness.Reason != "not ready" {
		t.Fatalf("Reason = %q, want not ready", readiness.Reason)
	}
	if len(readiness.Components) != 1 {
		t.Fatalf("Components length = %d, want 1", len(readiness.Components))
	}
	if readiness.Components[0].Name != "component" {
		t.Fatalf("Components[0].Name = %q, want component", readiness.Components[0].Name)
	}
}

func TestReadinessFromItemsWithNoItems(t *testing.T) {
	readiness := ReadinessFromItems("")

	if readiness.Ready {
		t.Fatal("Ready = true, want false")
	}
	if readiness.Reason != "no readiness items" {
		t.Fatalf("Reason = %q, want no readiness items", readiness.Reason)
	}
	if readiness.Components != nil {
		t.Fatalf("Components = %+v, want nil", readiness.Components)
	}
}

func TestReadinessFromItemsWithNoItemsPreservesReason(t *testing.T) {
	readiness := ReadinessFromItems("custom reason")

	if readiness.Ready {
		t.Fatal("Ready = true, want false")
	}
	if readiness.Reason != "custom reason" {
		t.Fatalf("Reason = %q, want custom reason", readiness.Reason)
	}
}

func TestReadinessFromItemsAllReady(t *testing.T) {
	items := []ReadinessItem{
		{Name: "cache", Ready: true},
		{Name: "database", Ready: true, State: StateDegraded},
	}

	readiness := ReadinessFromItems("", items...)
	items[0].Name = "mutated"

	if !readiness.Ready {
		t.Fatal("Ready = false, want true")
	}
	if readiness.Reason != "all readiness items ready" {
		t.Fatalf("Reason = %q, want all readiness items ready", readiness.Reason)
	}
	if readiness.Components[0].State != StateReady {
		t.Fatalf("Components[0].State = %q, want %q", readiness.Components[0].State, StateReady)
	}
	if readiness.Components[1].State != StateDegraded {
		t.Fatalf("Components[1].State = %q, want %q", readiness.Components[1].State, StateDegraded)
	}
	if readiness.Components[0].Name != "cache" {
		t.Fatalf("Components[0].Name = %q, want cache", readiness.Components[0].Name)
	}
}

func TestReadinessFromItemsNotReady(t *testing.T) {
	readiness := ReadinessFromItems("", ReadinessItem{Name: "cache", Ready: false})

	if readiness.Ready {
		t.Fatal("Ready = true, want false")
	}
	if readiness.Reason != "one or more readiness items are not ready" {
		t.Fatalf("Reason = %q, want one or more readiness items are not ready", readiness.Reason)
	}
	if readiness.Components[0].State != StateNotReady {
		t.Fatalf("Components[0].State = %q, want %q", readiness.Components[0].State, StateNotReady)
	}
}

func TestReadinessFromItemsPreservesReason(t *testing.T) {
	ready := ReadinessFromItems("custom ready", ReadinessItem{Name: "cache", Ready: true})
	if ready.Reason != "custom ready" {
		t.Fatalf("ready.Reason = %q, want custom ready", ready.Reason)
	}

	notReady := ReadinessFromItems("custom not ready", ReadinessItem{Name: "cache", Ready: false})
	if notReady.Reason != "custom not ready" {
		t.Fatalf("notReady.Reason = %q, want custom not ready", notReady.Reason)
	}
}

func TestReadinessFromStatusReady(t *testing.T) {
	readiness := ReadinessFromStatus(
		ComponentInfo{Name: "component", Kind: "test"},
		ReadyStatus("ready"),
	)

	if !readiness.Ready {
		t.Fatal("Ready = false, want true")
	}
	if readiness.Reason != "component ready" {
		t.Fatalf("Reason = %q, want component ready", readiness.Reason)
	}
	if len(readiness.Components) != 1 {
		t.Fatalf("Components length = %d, want 1", len(readiness.Components))
	}

	item := readiness.Components[0]
	if item.Name != "component" {
		t.Fatalf("Item.Name = %q, want component", item.Name)
	}
	if item.Kind != "test" {
		t.Fatalf("Item.Kind = %q, want test", item.Kind)
	}
	if !item.Ready {
		t.Fatal("Item.Ready = false, want true")
	}
	if item.State != StateReady {
		t.Fatalf("Item.State = %q, want %q", item.State, StateReady)
	}
	if item.Message != "ready" {
		t.Fatalf("Item.Message = %q, want ready", item.Message)
	}
}

func TestReadinessFromStatusNotReady(t *testing.T) {
	readiness := ReadinessFromStatus(
		ComponentInfo{Name: "component", Kind: "test"},
		NotReadyStatus("not ready"),
	)

	if readiness.Ready {
		t.Fatal("Ready = true, want false")
	}
	if readiness.Reason != "component not ready" {
		t.Fatalf("Reason = %q, want component not ready", readiness.Reason)
	}
	if len(readiness.Components) != 1 {
		t.Fatalf("Components length = %d, want 1", len(readiness.Components))
	}
	if readiness.Components[0].Ready {
		t.Fatal("Components[0].Ready = true, want false")
	}
	if readiness.Components[0].State != StateNotReady {
		t.Fatalf("Components[0].State = %q, want %q", readiness.Components[0].State, StateNotReady)
	}
}

func TestReadinessItemFromStatus(t *testing.T) {
	item := ReadinessItemFromStatus(
		ComponentInfo{Name: "component", Kind: "test"},
		DegradedStatus("degraded"),
	)

	if item.Name != "component" {
		t.Fatalf("Name = %q, want component", item.Name)
	}
	if item.Kind != "test" {
		t.Fatalf("Kind = %q, want test", item.Kind)
	}
	if !item.Ready {
		t.Fatal("Ready = false, want true")
	}
	if item.State != StateDegraded {
		t.Fatalf("State = %q, want %q", item.State, StateDegraded)
	}
	if item.Message != "degraded" {
		t.Fatalf("Message = %q, want degraded", item.Message)
	}
	if item.Policy != "" {
		t.Fatalf("Policy = %q, want empty policy", item.Policy)
	}
}

func TestReadinessItemFromStatusDefaultsEmptyStateFromReady(t *testing.T) {
	ready := ReadinessItemFromStatus(
		ComponentInfo{Name: "ready", Kind: "test"},
		Status{Ready: true, Message: "ready"},
	)
	if ready.State != StateReady {
		t.Fatalf("ready.State = %q, want %q", ready.State, StateReady)
	}

	notReady := ReadinessItemFromStatus(
		ComponentInfo{Name: "not-ready", Kind: "test"},
		Status{Ready: false, Message: "not ready"},
	)
	if notReady.State != StateNotReady {
		t.Fatalf("notReady.State = %q, want %q", notReady.State, StateNotReady)
	}
}

func TestCloneReadinessItems(t *testing.T) {
	items := []ReadinessItem{
		{Name: "component", Kind: "test", Ready: true, State: StateReady},
	}

	cloned := cloneReadinessItems(items)
	items[0].Name = "mutated"

	if len(cloned) != 1 {
		t.Fatalf("cloned length = %d, want 1", len(cloned))
	}
	if cloned[0].Name != "component" {
		t.Fatalf("cloned[0].Name = %q, want component", cloned[0].Name)
	}

	if got := cloneReadinessItems(nil); got != nil {
		t.Fatalf("cloneReadinessItems(nil) = %+v, want nil", got)
	}
	if got := cloneReadinessItems([]ReadinessItem{}); got != nil {
		t.Fatalf("cloneReadinessItems(empty) = %+v, want nil", got)
	}
}

func TestReadinessJSONOmitEmptyFields(t *testing.T) {
	requireJSON(t, Readiness{
		Ready: true,
	}, `{"ready":true}`)
}

func TestReadinessItemJSONIncludesPolicy(t *testing.T) {
	item := ReadinessItem{
		Name:    "component",
		Kind:    "test",
		Policy:  ReadinessOptional,
		Ready:   false,
		State:   StateNotReady,
		Reason:  "dependency unavailable",
		Message: "cache unavailable",
	}

	requireJSON(t, item, `{"name":"component","kind":"test","policy":"optional","ready":false,"state":"not_ready","reason":"dependency unavailable","message":"cache unavailable"}`)
}
