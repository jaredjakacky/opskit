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
	if len(readiness.Items) != 1 {
		t.Fatalf("Items length = %d, want 1", len(readiness.Items))
	}
	if readiness.Items[0].Name != "component" {
		t.Fatalf("Items[0].Name = %q, want component", readiness.Items[0].Name)
	}
	if readiness.Items[0].Impact != ReadinessImpactBlocking {
		t.Fatalf("Items[0].Impact = %q, want %q", readiness.Items[0].Impact, ReadinessImpactBlocking)
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
	if len(readiness.Items) != 1 {
		t.Fatalf("Items length = %d, want 1", len(readiness.Items))
	}
	if readiness.Items[0].Name != "component" {
		t.Fatalf("Items[0].Name = %q, want component", readiness.Items[0].Name)
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
	if readiness.Items != nil {
		t.Fatalf("Items = %+v, want nil", readiness.Items)
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
	if readiness.Reason != "all blocking readiness items ready" {
		t.Fatalf("Reason = %q, want all blocking readiness items ready", readiness.Reason)
	}
	if readiness.Items[0].State != StateReady {
		t.Fatalf("Items[0].State = %q, want %q", readiness.Items[0].State, StateReady)
	}
	if readiness.Items[1].State != StateDegraded {
		t.Fatalf("Items[1].State = %q, want %q", readiness.Items[1].State, StateDegraded)
	}
	if readiness.Items[0].Name != "cache" {
		t.Fatalf("Items[0].Name = %q, want cache", readiness.Items[0].Name)
	}
}

func TestReadinessFromItemsNotReady(t *testing.T) {
	readiness := ReadinessFromItems("", ReadinessItem{Name: "cache", Ready: false})

	if readiness.Ready {
		t.Fatal("Ready = true, want false")
	}
	if readiness.Reason != "one or more blocking readiness items are not ready" {
		t.Fatalf("Reason = %q, want one or more blocking readiness items are not ready", readiness.Reason)
	}
	if readiness.Items[0].State != StateNotReady {
		t.Fatalf("Items[0].State = %q, want %q", readiness.Items[0].State, StateNotReady)
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

func TestReadinessFromItemsAllBlockingReady(t *testing.T) {
	items := []ReadinessItem{
		{Name: "database", Impact: ReadinessImpactBlocking, Ready: true},
		{Name: "cache", Ready: true, State: StateDegraded},
	}

	readiness := ReadinessFromItems("", items...)
	items[0].Name = "mutated"

	if !readiness.Ready {
		t.Fatal("Ready = false, want true")
	}
	if readiness.Reason != "all blocking readiness items ready" {
		t.Fatalf("Reason = %q, want all blocking readiness items ready", readiness.Reason)
	}
	if readiness.Items[0].Name != "database" {
		t.Fatalf("Items[0].Name = %q, want database", readiness.Items[0].Name)
	}
	if readiness.Items[0].State != StateReady {
		t.Fatalf("Items[0].State = %q, want %q", readiness.Items[0].State, StateReady)
	}
	if readiness.Items[1].Impact != ReadinessImpactBlocking {
		t.Fatalf("Items[1].Impact = %q, want %q", readiness.Items[1].Impact, ReadinessImpactBlocking)
	}
	if readiness.Items[1].State != StateDegraded {
		t.Fatalf("Items[1].State = %q, want %q", readiness.Items[1].State, StateDegraded)
	}
}

func TestReadinessFromItemsBlockingNotReadyBlocks(t *testing.T) {
	readiness := ReadinessFromItems("", ReadinessItem{
		Name:   "database",
		Impact: ReadinessImpactBlocking,
		Ready:  false,
	})

	if readiness.Ready {
		t.Fatal("Ready = true, want false")
	}
	if readiness.Reason != "one or more blocking readiness items are not ready" {
		t.Fatalf("Reason = %q, want one or more blocking readiness items are not ready", readiness.Reason)
	}
	if readiness.Items[0].State != StateNotReady {
		t.Fatalf("Items[0].State = %q, want %q", readiness.Items[0].State, StateNotReady)
	}
}

func TestReadinessFromItemsNonBlockingItemsDoNotBlock(t *testing.T) {
	readiness := ReadinessFromItems("",
		ReadinessItem{Name: "database", Impact: ReadinessImpactBlocking, Ready: true},
		ReadinessItem{Name: "cache", Impact: ReadinessImpactNonBlocking, Ready: false},
		ReadinessItem{Name: "search", Impact: ReadinessImpactNonBlocking, Ready: false},
	)

	if !readiness.Ready {
		t.Fatal("Ready = false, want true")
	}
	if len(readiness.Items) != 3 {
		t.Fatalf("Items length = %d, want 3", len(readiness.Items))
	}
	if readiness.Items[1].Impact != ReadinessImpactNonBlocking {
		t.Fatalf("Items[1].Impact = %q, want %q", readiness.Items[1].Impact, ReadinessImpactNonBlocking)
	}
	if readiness.Items[2].Impact != ReadinessImpactNonBlocking {
		t.Fatalf("Items[2].Impact = %q, want %q", readiness.Items[2].Impact, ReadinessImpactNonBlocking)
	}
}

func TestReadinessFromItemsWithoutBlockingItemsIsReady(t *testing.T) {
	readiness := ReadinessFromItems("",
		ReadinessItem{Name: "cache", Impact: ReadinessImpactNonBlocking, Ready: true},
		ReadinessItem{Name: "search", Impact: ReadinessImpactNonBlocking, Ready: false},
	)

	if !readiness.Ready {
		t.Fatal("Ready = false, want true")
	}
	if readiness.Reason != "no blocking readiness items" {
		t.Fatalf("Reason = %q, want no blocking readiness items", readiness.Reason)
	}
	if len(readiness.Items) != 2 {
		t.Fatalf("Items length = %d, want 2", len(readiness.Items))
	}
}

func TestReadinessFromItemsWithoutBlockingItemsPreservesReason(t *testing.T) {
	readiness := ReadinessFromItems("optional clients do not block",
		ReadinessItem{Name: "search", Impact: ReadinessImpactNonBlocking, Ready: false},
	)

	if !readiness.Ready {
		t.Fatal("Ready = false, want true")
	}
	if readiness.Reason != "optional clients do not block" {
		t.Fatalf("Reason = %q, want optional clients do not block", readiness.Reason)
	}
}

func TestReadinessFromItemsDefaultsUnknownImpactToBlocking(t *testing.T) {
	readiness := ReadinessFromItems("", ReadinessItem{
		Name:   "database",
		Impact: ReadinessImpact("unknown"),
		Ready:  true,
	})

	if !readiness.Ready {
		t.Fatal("Ready = false, want true")
	}
	if readiness.Items[0].Impact != ReadinessImpactBlocking {
		t.Fatalf("Items[0].Impact = %q, want %q", readiness.Items[0].Impact, ReadinessImpactBlocking)
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
	if len(readiness.Items) != 1 {
		t.Fatalf("Items length = %d, want 1", len(readiness.Items))
	}

	item := readiness.Items[0]
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
	if len(readiness.Items) != 1 {
		t.Fatalf("Items length = %d, want 1", len(readiness.Items))
	}
	if readiness.Items[0].Ready {
		t.Fatal("Items[0].Ready = true, want false")
	}
	if readiness.Items[0].State != StateNotReady {
		t.Fatalf("Items[0].State = %q, want %q", readiness.Items[0].State, StateNotReady)
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
	if item.Impact != ReadinessImpactBlocking {
		t.Fatalf("Impact = %q, want %q", item.Impact, ReadinessImpactBlocking)
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

func TestReadinessJSONOmitEmptyFields(t *testing.T) {
	requireJSON(t, Readiness{
		Ready: true,
	}, `{"ready":true}`)
}

func TestSystemReadinessJSONPreservesParentAndChildSemantics(t *testing.T) {
	requireJSON(t, SystemReadiness{
		Ready:  true,
		Reason: "service ready",
		Components: []ComponentReadiness{
			{
				Component: ComponentInfo{Name: "clients", Kind: "client_registry"},
				Registration: ComponentRegistration{
					ReadinessPolicy: ReadinessOptional,
				},
				Readiness: Readiness{
					Ready:  false,
					Reason: "payments unavailable",
					Items: []ReadinessItem{
						{
							Name:   "payments",
							Impact: ReadinessImpactBlocking,
							Ready:  false,
							State:  StateNotReady,
						},
					},
				},
			},
		},
	}, `{"ready":true,"reason":"service ready","components":[{"component":{"name":"clients","kind":"client_registry"},"registration":{"readiness_policy":"optional"},"readiness":{"ready":false,"reason":"payments unavailable","items":[{"name":"payments","impact":"blocking","ready":false,"state":"not_ready"}]}}]}`)
}

func TestReadinessItemJSONIncludesImpact(t *testing.T) {
	item := ReadinessItem{
		Name:    "component",
		Kind:    "test",
		Impact:  ReadinessImpactNonBlocking,
		Ready:   false,
		State:   StateNotReady,
		Reason:  "dependency unavailable",
		Message: "cache unavailable",
	}

	requireJSON(t, item, `{"name":"component","kind":"test","impact":"non_blocking","ready":false,"state":"not_ready","reason":"dependency unavailable","message":"cache unavailable"}`)
}
